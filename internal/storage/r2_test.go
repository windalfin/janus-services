package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// mockS3Client implements s3ClientAPI interface for testing
type mockS3Client struct {
	failCount    int
	currentCount int
	failureDelay time.Duration
	t            *testing.T
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	// Simulate some delay
	time.Sleep(m.failureDelay)

	// Check if context is cancelled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.t.Logf("Upload attempt %d of %d", m.currentCount+1, m.failCount+1)

	if m.currentCount < m.failCount {
		m.currentCount++
		return nil, fmt.Errorf("simulated failure %d of %d", m.currentCount, m.failCount)
	}

	return &s3.PutObjectOutput{}, nil
}

func TestR2ClientWithMock(t *testing.T) {
	tests := []struct {
		name        string
		failCount   int
		delay       time.Duration
		shouldPass  bool
		timeoutCtx  bool
		inputSize   int64
		description string
	}{
		{
			name:        "Success after 2 retries",
			failCount:   2,
			delay:       50 * time.Millisecond,
			shouldPass:  true,
			timeoutCtx:  false,
			inputSize:   100,
			description: "Should succeed after 2 retries (3 attempts total)",
		},
		{
			name:        "Immediate success",
			failCount:   0,
			delay:       0,
			shouldPass:  true,
			timeoutCtx:  false,
			inputSize:   100,
			description: "Should succeed on first attempt",
		},
		{
			name:        "Fail after max retries",
			failCount:   5,
			delay:       50 * time.Millisecond,
			shouldPass:  false,
			timeoutCtx:  false,
			inputSize:   100,
			description: "Should fail after exceeding max retry attempts",
		},
		{
			name:        "Context timeout",
			failCount:   1,
			delay:       200 * time.Millisecond,
			shouldPass:  false,
			timeoutCtx:  true,
			inputSize:   100,
			description: "Should fail due to context timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data
			testData := strings.NewReader(strings.Repeat("a", int(tt.inputSize)))

			// Create context
			var ctx context.Context
			var cancel context.CancelFunc
			if tt.timeoutCtx {
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			}
			defer cancel()

			// Create mock client
			mockClient := &mockS3Client{
				failCount:    tt.failCount,
				failureDelay: tt.delay,
				t:            t,
			}

			r2Client := &R2Client{
				client: mockClient,
				bucket: "test-bucket",
			}

			// Attempt upload
			url, err := r2Client.Upload(ctx, "test-key", testData, "text/plain")

			// Verify results
			if tt.shouldPass {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if url == "" {
					t.Error("Expected URL to be returned, got empty string")
				}
			} else {
				if err == nil {
					t.Error("Expected error, got success")
				}
			}

			// Verify retry count matches expectations
			if tt.shouldPass && mockClient.currentCount != tt.failCount {
				t.Errorf("Expected %d retries, got %d", tt.failCount, mockClient.currentCount)
			}
		})
	}
}
