package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start(port)
	}()

	log.Printf("Server running on port %s", port)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrCh:
		if err != nil {
			log.Fatalf("server failed: %v", err)
		}
		return
	case sig := <-sigCh:
		log.Printf("received signal %s, shutting down", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	if err := <-serverErrCh; err != nil {
		log.Printf("server stopped with error: %v", err)
	}
}
