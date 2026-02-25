package main

import (
	"context"
	"errors"
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
	log.Println("phase=startup step=load_env")
	env, err := appenv.Load()
	if err != nil {
		log.Fatalf("failed to load environment: %v", err)
	}
	log.Printf("phase=startup step=env_loaded app_env=%s port=%s", env.AppEnv, env.ListenAddr())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("phase=startup step=open_database")
	database, err := db.New(db.Config{
		DatabaseURI: env.DatabaseURI,
		Environment: env.AppEnv,
	})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	log.Println("phase=startup step=database_opened")

	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	log.Println("phase=startup step=run_migrations")
	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	log.Println("phase=startup step=migrations_complete")

	log.Println("phase=startup step=init_server")
	srv, err := server.New(
		server.WithDatabase(database),
		server.WithPublicFS(os.DirFS("public")),
	)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}
	log.Println("phase=startup step=server_initialized")

	_ = settings.New(srv)

	log.Println("phase=startup step=register_routes")
	routes.RegisterServices(srv)
	log.Println("phase=startup step=routes_registered")

	listenAddr := env.ListenAddr()
	log.Printf("phase=startup step=listen addr=http://localhost%s", listenAddr)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Listen(listenAddr)
	}()

	select {
	case err := <-serveErr:
		if err != nil && !isExpectedServerCloseError(err) {
			log.Fatalf("server stopped with error: %v", err)
		}
		log.Println("phase=shutdown step=server_stopped")
	case <-ctx.Done():
		log.Printf("phase=shutdown step=signal_received signal=%v", ctx.Err())
		shutdownDone := make(chan error, 1)
		shutdownTimeout := env.AppShutdownTimeout
		go func() {
			log.Println("phase=shutdown step=server_shutdown_start")
			shutdownDone <- srv.Shutdown()
		}()

		select {
		case err := <-shutdownDone:
			if err != nil && !errors.Is(err, context.Canceled) && !isExpectedServerCloseError(err) {
				log.Printf("graceful shutdown error: %v", err)
			} else {
				log.Println("phase=shutdown step=server_shutdown_done")
			}
		case <-time.After(shutdownTimeout):
			log.Printf("phase=shutdown step=shutdown_timeout duration=%s", shutdownTimeout)
		}

		select {
		case err := <-serveErr:
			if err != nil && !isExpectedServerCloseError(err) {
				log.Printf("phase=shutdown step=server_exit_error err=%v", err)
			} else {
				log.Println("phase=shutdown step=server_stopped")
			}
		case <-time.After(shutdownTimeout):
			log.Printf("phase=shutdown step=listen_exit_timeout duration=%s", shutdownTimeout)
		}
	}

	log.Println("phase=shutdown step=complete")
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
