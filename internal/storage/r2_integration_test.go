//go:build integration

package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"janus-services/internal/config"
)

// TestR2Integration performs actual uploads to R2
// To run this test, use: go test -v -tags=integration

func TestR2Integration(t *testing.T) {
	// Load .env file
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal("Error loading .env file:", err)
	}

	// Create config
	cfg := config.NewConfig()
	if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2AccessKeySecret == "" || cfg.R2Bucket == "" {
		t.Fatal("Missing required R2 credentials in environment")
	}

	// Create R2 client
	r2Client, err := NewR2Client(cfg)
	if err != nil {
		t.Fatal("Failed to create R2 client:", err)
	}

	tests := []struct {
		name        string
		content     string
		extension   string
		expectError bool
	}{
		{
			name:        "Upload text file",
			content:     "This is a test file for R2 upload",
			extension:   ".txt",
			expectError: false,
		},
		{
			name:        "Upload markdown file",
			content:     "# Test Markdown\nThis is a test markdown file.",
			extension:   ".md",
			expectError: false,
		},
		{
			name:        "Upload empty file",
			content:     "",
			extension:   ".empty",
			expectError: false,
		},
	}

	testDir := "../../test_files"
	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal("Failed to create test directory:", err)
	}
	defer os.RemoveAll(testDir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			fileName := time.Now().Format("20060102_150405") + tt.extension
			filePath := filepath.Join(testDir, fileName)
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatal("Failed to create test file:", err)
			}

			// Upload file
			url, err := r2Client.UploadFile(filePath)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got success")
				}
				return
			}

			if err != nil {
				t.Fatal("Upload failed:", err)
			}

			t.Logf("Successfully uploaded %s to R2. URL: %s", fileName, url)

			// Verify the URL format
			expectedPrefix := "https://" + cfg.R2Bucket + ".r2.cloudflarestorage.com/recordings/"
			if url[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("Expected URL to start with %s, got %s", expectedPrefix, url)
			}
		})
	}
}
