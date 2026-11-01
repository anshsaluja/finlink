package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/anshsaluja/finlink/internal/httpserver"
)

func main() {
	addr := envOrDefault("FINLINK_HTTP_ADDR", ":8080")

	server := &http.Server{
		Addr:              addr,
		Handler:           httpserver.New(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("finlink api listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
