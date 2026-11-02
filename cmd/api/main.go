package main

import (
	"log"
	"net/http"
	"time"

	"github.com/anshsaluja/finlink/internal/config"
	"github.com/anshsaluja/finlink/internal/httpserver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.New(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("finlink api listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
