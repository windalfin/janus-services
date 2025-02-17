package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	RecordingsPath    string
	ProcessedPath     string
	DatabasePath      string
	JanusPPRecPath    string
	R2Bucket          string
	R2AccountID       string
	R2AccessKeyID     string
	R2AccessKeySecret string
}

func NewConfig() *Config {
	// Default paths
	recordingsPath := "/recordings" // Inside container
	processedPath := filepath.Join(recordingsPath, "processed")
	dbPath := "/var/log/janus-processor/processing.db"
	janusPPRecPath := os.Getenv("JANUS_PP_REC_PATH")

	// Override with environment variables if provided
	if envPath := os.Getenv("JANUS_RECORDINGS_PATH"); envPath != "" {
		recordingsPath = envPath
	}
	if envPath := os.Getenv("JANUS_PROCESSED_PATH"); envPath != "" {
		processedPath = envPath
	}
	if envPath := os.Getenv("JANUS_DB_PATH"); envPath != "" {
		dbPath = envPath
	}

	return &Config{
		RecordingsPath:    recordingsPath,
		ProcessedPath:     processedPath,
		DatabasePath:      dbPath,
		JanusPPRecPath:    janusPPRecPath,
		R2Bucket:          os.Getenv("R2_BUCKET"),
		R2AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		R2AccessKeySecret: os.Getenv("R2_ACCESS_KEY_SECRET"),
	}
}
