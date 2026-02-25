package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sensorpanel/internal/appenv"
	"sensorpanel/internal/db"
	"sensorpanel/internal/routes"
	"sensorpanel/internal/server"
	"sensorpanel/internal/services/settings"
)

func main() {
	fmt.Println("🔧  Loading Env...")
	env, err := appenv.Load()
	if err != nil {
		log.Fatalf("failed to load environment: %v", err)
	}

	defer func() {
		fmt.Println("✅ All cleanup tasks completed")
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("🗄️   Opening Database...")
	database, err := db.New(db.Config{
		DatabaseURI: env.DatabaseURI,
		Environment: env.AppEnv,
	})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	defer func() {
		fmt.Println("🔌 Closing database connections...")
		if err := database.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	fmt.Println("🪿  Running Migrations...")
	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	fmt.Println("🚀  Initializing Server...")
	s, err := server.New(
		server.WithDatabase(database),
		server.WithPublicFS(os.DirFS("public")),
	)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	_ = settings.New(s)

	routes.RegisterServices(s)

	listenAddr := env.ListenAddr()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.Listen(listenAddr)
	}()

	select {
	case err := <-serveErr:
		if err != nil && !isExpectedServerCloseError(err) {
			log.Fatalf("server stopped with error: %v", err)
		}
	case <-ctx.Done():
		fmt.Println("🩰 Graceful shutdown requested...")
		shutdownDone := make(chan error, 1)
		shutdownTimeout := env.AppShutdownTimeout
		go func() {
			shutdownDone <- s.Shutdown()
		}()

		select {
		case err := <-shutdownDone:
			if err != nil && !errors.Is(err, context.Canceled) && !isExpectedServerCloseError(err) {
				log.Printf("graceful shutdown error: %v", err)
			} else {
				fmt.Println("✅  Server shutdown completed")
			}
		case <-time.After(shutdownTimeout):
			log.Printf("graceful shutdown timed out after %s", shutdownTimeout)
		}

		select {
		case err := <-serveErr:
			if err != nil && !isExpectedServerCloseError(err) {
				log.Printf("server exit error after shutdown: %v", err)
			}
		case <-time.After(shutdownTimeout):
			log.Printf("listen exit timed out after %s", shutdownTimeout)
		}
	}

	fmt.Println("🧹 Running cleanup tasks...")
}

func isExpectedServerCloseError(err error) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "server closed") ||
		strings.Contains(message, "listener closed") ||
		strings.Contains(message, "use of closed network connection")
}
