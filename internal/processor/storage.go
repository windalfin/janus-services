package processor

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
)

// UploadToR2 uploads a file to R2 Storage with retry logic
func UploadToR2(ctx context.Context, client *s3.Client, bucket string, key string, data io.Reader, contentType string) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// If this is a retry, we need to reset the reader to the beginning if possible
		if attempt > 0 {
			if seeker, ok := data.(io.Seeker); ok {
				_, err := seeker.Seek(0, io.SeekStart)
				if err != nil {
					return fmt.Errorf("failed to reset reader for retry: %w", err)
				}
			} else {
				return fmt.Errorf("cannot retry upload: reader is not seekable and previous attempt failed: %w", lastErr)
			}

			// Calculate exponential backoff delay
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		input := &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        data,
			ContentType: aws.String(contentType),
			Expires:     aws.Time(time.Now().Add(24 * time.Hour * 365)), // 1 year expiration
		}

		_, err := client.PutObject(ctx, input)
		if err == nil {
			return nil // Success!
		}

		lastErr = err
		// Log retry attempt if not the last try
		if attempt < maxRetries-1 {
			fmt.Printf("Upload attempt %d failed: %v. Retrying...\n", attempt+1, err)
		}
	}

	return fmt.Errorf("upload failed after %d attempts. Last error: %w", maxRetries, lastErr)
}
