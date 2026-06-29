package gateway

import (
	"bytes"
	"context"
	"encoding/json"
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
			return nil, NewAPIError(resp.StatusCode, bodyBytes)
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

// APIError represents a structured error returned by upstream LLM APIs.
type APIError struct {
	StatusCode int
	Message    string
	Code       string
	Status     string
	ErrorType  string
	RetryAfter string
	RawBody    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API request failed with status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API request failed with status %d", e.StatusCode)
}

// NewAPIError parses raw response bodies from Gemini, OpenAI, and Anthropic APIs into a structured APIError.
func NewAPIError(statusCode int, bodyBytes []byte) error {
	apiErr := &APIError{
		StatusCode: statusCode,
		RawBody:    string(bodyBytes),
	}

	var payload struct {
		Error *struct {
			Message string           `json:"message"`
			Status  string           `json:"status"`
			Code    any              `json:"code"`
			Type    string           `json:"type"`
			Details []map[string]any `json:"details"`
		} `json:"error"`
		Type    string `json:"type"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(bodyBytes, &payload); err == nil {
		if payload.Error != nil {
			apiErr.Message = payload.Error.Message
			if payload.Error.Code != nil {
				apiErr.Code = fmt.Sprintf("%v", payload.Error.Code)
			}
			apiErr.Status = payload.Error.Status
			apiErr.ErrorType = payload.Error.Type

			// Extract Retry delay if available (Gemini retryDelay)
			for _, detail := range payload.Error.Details {
				if detail["@type"] == "type.googleapis.com/google.rpc.RetryInfo" {
					if delay, ok := detail["retryDelay"].(string); ok {
						apiErr.RetryAfter = delay
					}
				}
			}
		} else if payload.Message != "" {
			// Anthropic or other flat error structures
			apiErr.Message = payload.Message
			apiErr.ErrorType = payload.Type
		}
	}

	if apiErr.Message == "" {
		// Pretty-print valid JSON if possible, otherwise fallback to raw string
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, bodyBytes, "", "  "); err == nil {
			apiErr.Message = formatted.String()
		} else {
			apiErr.Message = string(bodyBytes)
		}
	}

	return apiErr
}
