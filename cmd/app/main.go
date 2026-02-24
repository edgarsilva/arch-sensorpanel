package main

import (
	"log"
	"os"

	"sensorpanel/internal/db"
	"sensorpanel/internal/routes"
	"sensorpanel/internal/server"
	"sensorpanel/internal/services/settings"
)

func main() {
	database, err := db.New(db.Config{
		DatabaseURI: os.Getenv("DATABASE_URI"),
		Environment: os.Getenv("APP_ENV"),
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

	log.Println("Listening on http://localhost:9070")
	log.Fatal(srv.Listen(":9070"))
}
