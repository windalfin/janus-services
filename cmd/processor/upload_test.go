package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
	processor "janus-services/internal/processor"
)

func TestUploadToR2(t *testing.T) {
	// Load .env file
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal("Error loading .env file:", err)
	}

	// Get R2 credentials from environment
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("R2_ACCESS_KEY_SECRET")
	bucketName := os.Getenv("R2_BUCKET")

	if accountID == "" || accessKeyID == "" || secretAccessKey == "" || bucketName == "" {
		t.Fatal("Missing required R2 credentials in environment")
	}

	// Create R2 client
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("auto"),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	)
	if err != nil {
		t.Fatal("Unable to load SDK config:", err)
	}

	client := s3.NewFromConfig(cfg)

	// Open test file
	file, err := os.Open("../../data/processed/sample_data.md")
	if err != nil {
		t.Fatal("Error opening test file:", err)
	}
	defer file.Close()

	// Upload file
	err = processor.UploadToR2(
		context.Background(),
		client,
		bucketName,
		"processed/sample_data.md",
		file,
		"text/markdown",
	)

	if err != nil {
		t.Fatal("Error uploading to R2:", err)
	}

	t.Log("Successfully uploaded sample_data.md to R2")
}
