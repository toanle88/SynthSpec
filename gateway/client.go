package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// SendWithRetry executes an HTTP request with exponential backoff on transient errors (429, 5xx).
func SendWithRetry(ctx context.Context, client *http.Client, req *http.Request, maxRetries int) ([]byte, error) {
	var reqBody []byte
	var err error

	// Cache request body if we need to replay it
	if req.Body != nil {
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Restore request body for this attempt
		if reqBody != nil {
			req.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		// Check context cancelation
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		resp, err := client.Do(req.WithContext(ctx))
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("network error after %d retries: %w", maxRetries, err)
			}
			waitBackoff(ctx, attempt)
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return bodyBytes, nil
		}

		// Check if we should retry
		shouldRetry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500

		if !shouldRetry || attempt == maxRetries {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		waitBackoff(ctx, attempt)
	}

	return nil, fmt.Errorf("request failed after %d retries", maxRetries)
}

func waitBackoff(ctx context.Context, attempt int) {
	backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
	if backoff > 16*time.Second {
		backoff = 16 * time.Second
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
