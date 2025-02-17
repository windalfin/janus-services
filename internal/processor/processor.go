package processor

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"janus-services/internal/config"
	"janus-services/internal/db"
	"janus-services/internal/storage"
)

type Processor struct {
	config   *config.Config
	db       *sql.DB
	r2       *storage.R2Client
	janusCmd string
}

func NewProcessor(cfg *config.Config, db *sql.DB, r2 *storage.R2Client, janusCmd string) *Processor {
	if janusCmd == "" {
		janusCmd = "/usr/local/bin/janus-pp-rec" // Default path on host
	}
	return &Processor{
		config:   cfg,
		db:       db,
		r2:       r2,
		janusCmd: janusCmd,
	}
}

func (p *Processor) ConvertMJRToOpus(inputPath string) (string, error) {
	// Create output path with .opus extension
	outputPath := strings.TrimSuffix(inputPath, ".mjr") + ".opus"

	// Use absolute paths
	absInputPath, err := filepath.Abs(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute input path: %w", err)
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute output path: %w", err)
	}

	// Execute janus-pp-rec on host machine
	cmd := exec.Command(p.janusCmd, absInputPath, absOutputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("conversion failed: %w, output: %s", err, string(output))
	}

	// Verify the output file was created
	if _, err := os.Stat(absOutputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("conversion completed but output file not found: %s", absOutputPath)
	}

	log.Printf("Successfully converted %s to %s", absInputPath, absOutputPath)
	return absOutputPath, nil
}

func (p *Processor) MoveToProcessed(sourcePath string) error {
	fileName := filepath.Base(sourcePath)
	destPath := filepath.Join(p.config.ProcessedPath, fileName)

	// Ensure the processed directory exists
	if err := os.MkdirAll(p.config.ProcessedPath, 0755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}

	return os.Rename(sourcePath, destPath)
}

func (p *Processor) ProcessFile(filePath string) error {
	log := db.ProcessingLog{
		SourceFile:  filePath,
		ProcessedAt: time.Now(),
		Status:      "processing",
	}

	// Convert MJR to Opus
	opusFile, err := p.ConvertMJRToOpus(filePath)
	if err != nil {
		log.Status = "error"
		log.Error = err.Error()
		return db.LogProcess(p.db, log)
	}
	log.OutputFile = opusFile

	// Upload to R2
	r2URL, err := p.r2.UploadFile(opusFile)
	if err != nil {
		log.Status = "error"
		log.Error = err.Error()
		return db.LogProcess(p.db, log)
	}
	log.R2URL = r2URL

	// Move original file to processed folder
	if err := p.MoveToProcessed(filePath); err != nil {
		log.Status = "error"
		log.Error = err.Error()
		return db.LogProcess(p.db, log)
	}

	log.Status = "completed"
	return db.LogProcess(p.db, log)
}

func (p *Processor) Start() error {
	// Create necessary directories
	if err := os.MkdirAll(p.config.ProcessedPath, 0755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}

	// Process files
	pattern := filepath.Join(p.config.RecordingsPath, "*-audio.mjr")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list MJR files: %w", err)
	}

	for _, file := range files {
		if err := p.ProcessFile(file); err != nil {
			fmt.Printf("Error processing file %s: %v\n", file, err)
			continue
		}
		fmt.Printf("Successfully processed file: %s\n", file)
	}

	return nil
}
