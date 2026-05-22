package main

import (
	"fmt"
	"log"
	"os"

	"spendsense-backend/internal/httpapi"

	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(".env"); err != nil {
		fmt.Printf("No .env loaded: %v\n", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")	
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	server, err := httpapi.NewServer(databaseURL, jwtSecret)
	if err != nil {
		log.Fatalf("failed to initialize API: %v", err)
	}

	if err := server.Start(port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
