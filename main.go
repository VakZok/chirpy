package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/VakZok/chirpy/internal/api"
	"github.com/VakZok/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Error connecting to DB: %s", err)
		return
	}

	dbQueries := database.New(db)

	// Load environmental variables
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM must be set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	cfg := api.NewConfig(dbQueries, platform, jwtSecret)

	// configuring http server
	s := &http.Server{
		Addr:    ":8080",
		Handler: api.NewRouter(cfg),
	}

	// start server and have it continously run listening for requests to serve
	log.Fatal(s.ListenAndServe())
}
