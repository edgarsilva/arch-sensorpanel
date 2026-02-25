package main

import (
	"log"
	"os"

	"sensorpanel/internal/appenv"
	"sensorpanel/internal/db"
	"sensorpanel/internal/routes"
	"sensorpanel/internal/server"
	"sensorpanel/internal/services/settings"
)

func main() {
	env, err := appenv.Load()
	if err != nil {
		log.Fatalf("failed to load environment: %v", err)
	}

	database, err := db.New(db.Config{
		DatabaseURI: env.DatabaseURI,
		Environment: env.AppEnv,
	})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	srv, err := server.New(
		server.WithDatabase(database),
		server.WithPublicFS(os.DirFS("public")),
	)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	_ = settings.New(srv)

	routes.RegisterServices(srv)

	log.Printf("Listening on http://localhost%s", env.ListenAddr())
	log.Fatal(srv.Listen(env.ListenAddr()))
}
