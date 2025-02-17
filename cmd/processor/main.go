package main

import (
	"flag"
	"log"
	"net/http"

	"janus-services/internal/config"
	"janus-services/internal/db"
	"janus-services/internal/processor"
	"janus-services/internal/storage"
)

func setupHealthCheck() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("Health check server error: %v", err)
		}
	}()
}

func main() {
	// Command line flags
	janusCmd := flag.String("janus-cmd", "janus-pp-rec", "Path to janus-pp-rec executable")
	flag.Parse()

	// Setup health check endpoint
	setupHealthCheck()

	// Initialize config
	cfg := config.NewConfig()

	// Initialize database
	database, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize R2 client
	r2Client, err := storage.NewR2Client(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize R2 client: %v", err)
	}

	// Create processor instance
	proc := processor.NewProcessor(cfg, database, r2Client, *janusCmd)

	// Start processing
	if err := proc.Start(); err != nil {
		log.Fatalf("Processor failed: %v", err)
	}
}
