package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"janus-services/internal/config"
)

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
)

// s3ClientAPI interface for mocking in tests
type s3ClientAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type R2Client struct {
	client s3ClientAPI
	bucket string
}

func NewR2Client(cfg *config.Config) (*R2Client, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID),
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithEndpointResolverWithOptions(r2Resolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.R2AccessKeyID,
			cfg.R2AccessKeySecret,
			"",
		)),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg)
	return &R2Client{
		client: client,
		bucket: cfg.R2Bucket,
	}, nil
}

// Upload uploads any data to R2 with retry logic
func (r *R2Client) Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// If this is a retry, we need to reset the reader to the beginning if possible
		if attempt > 0 {
			if seeker, ok := data.(io.Seeker); ok {
				_, err := seeker.Seek(0, io.SeekStart)
				if err != nil {
					return "", fmt.Errorf("failed to reset reader for retry: %w", err)
				}
			} else {
				return "", fmt.Errorf("cannot retry upload: reader is not seekable and previous attempt failed: %w", lastErr)
			}

			// Calculate exponential backoff delay
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		input := &s3.PutObjectInput{
			Bucket:      aws.String(r.bucket),
			Key:         aws.String(key),
			Body:        data,
			ContentType: aws.String(contentType),
		}

		_, err := r.client.PutObject(ctx, input)
		if err == nil {
			return fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s", r.bucket, key), nil
		}

		lastErr = err
		if attempt < maxRetries-1 {
			fmt.Printf("Upload attempt %d failed: %v. Retrying...\n", attempt+1, err)
		}
	}

	return "", fmt.Errorf("upload failed after %d attempts. Last error: %w", maxRetries, lastErr)
}

// UploadFile uploads a file to R2 with retry logic
func (r *R2Client) UploadFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileName := filepath.Base(filePath)
	key := fmt.Sprintf("recordings/%s", fileName)

	// Use the generic Upload method with context and content type
	ctx := context.Background()
	contentType := "application/octet-stream" // You might want to detect this based on file extension
	return r.Upload(ctx, key, file, contentType)
}
