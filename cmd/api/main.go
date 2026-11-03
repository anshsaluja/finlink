package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/anshsaluja/finlink/internal/config"
	"github.com/anshsaluja/finlink/internal/connections"
	"github.com/anshsaluja/finlink/internal/financialdata"
	"github.com/anshsaluja/finlink/internal/httpserver"
	"github.com/anshsaluja/finlink/internal/kafkaevents"
	"github.com/anshsaluja/finlink/internal/provider"
	"github.com/anshsaluja/finlink/internal/syncs"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	registry := provider.NewRegistry(provider.MockNorthstar{}, provider.MockHarbor{})
	connectionService := connections.NewServiceWithRepository(registry, connections.NewSQLRepository(db))
	publisher := kafkaevents.NewPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Printf("close kafka publisher: %v", err)
		}
	}()
	syncService := syncs.NewService(syncs.NewSQLRepository(db), connectionService, publisher)
	financialDataRepository := financialdata.NewSQLRepository(db)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.NewWithServicesAndData(connectionService, syncService, financialDataRepository),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("finlink api listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
