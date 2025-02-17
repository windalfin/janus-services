package main

import (
	"flag"
	"log"
	"net/http"
	"time"

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
}

func main() {
	// Command line flags
	janusCmd := flag.String("janus-cmd", "janus-pp-rec", "Path to janus-pp-rec executable")
	interval := flag.Int("interval", 60, "Processing interval in seconds")
	flag.Parse()

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

	// Setup health check endpoint
	setupHealthCheck()

	// Setup manual trigger endpoint
	http.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		if err := proc.Start(); err != nil {
			log.Printf("Manual processing error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Processing completed"))
	})

	// Start periodic processing
	go func() {
		ticker := time.NewTicker(time.Duration(*interval) * time.Second)
		defer ticker.Stop()

		for {
			if err := proc.Start(); err != nil {
				log.Printf("Periodic processing error: %v", err)
			}
			<-ticker.C
		}
	}()

	// Start HTTP server
	log.Printf("Server started. Health check on :8080/health, Manual processing on :8080/process")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
