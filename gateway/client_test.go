package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAPIError_Gemini(t *testing.T) {
	rawJSON := `{
		"error": {
			"code": 429,
			"message": "You exceeded your current quota, please check your plan and billing details.",
			"status": "RESOURCE_EXHAUSTED",
			"details": [
				{
					"@type": "type.googleapis.com/google.rpc.Help",
					"links": [
						{
							"description": "Learn more about Gemini API quotas",
							"url": "https://ai.google.dev/gemini-api/docs/rate-limits"
						}
					]
				},
				{
					"@type": "type.googleapis.com/google.rpc.QuotaFailure",
					"violations": [
						{
							"quotaMetric": "generativelanguage.googleapis.com/generate_content_free_tier_requests",
							"quotaId": "GenerateRequestsPerDayPerProjectPerModel-FreeTier",
							"quotaDimensions": {
								"location": "global",
								"model": "gemini-3-flash"
							},
							"quotaValue": "20"
						}
					]
				},
				{
					"@type": "type.googleapis.com/google.rpc.RetryInfo",
					"retryDelay": "38s"
				}
			]
		}
	}`

	err := NewAPIError(429, []byte(rawJSON))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected error to be *APIError, got %T", err)
	}

	if apiErr.StatusCode != 429 {
		t.Errorf("expected status code 429, got %d", apiErr.StatusCode)
	}
	if apiErr.Status != "RESOURCE_EXHAUSTED" {
		t.Errorf("expected status 'RESOURCE_EXHAUSTED', got %q", apiErr.Status)
	}
	if apiErr.Code != "429" {
		t.Errorf("expected code '429', got %q", apiErr.Code)
	}
	if apiErr.Message != "You exceeded your current quota, please check your plan and billing details." {
		t.Errorf("unexpected message: %q", apiErr.Message)
	}
	if apiErr.RetryAfter != "38s" {
		t.Errorf("expected retry delay '38s', got %q", apiErr.RetryAfter)
	}
}

func TestNewAPIError_OpenAI(t *testing.T) {
	rawJSON := `{
		"error": {
			"message": "You exceeded your current quota, please check your plan and billing details.",
			"type": "insufficient_quota",
			"param": null,
			"code": "insufficient_quota"
		}
	}`

	err := NewAPIError(429, []byte(rawJSON))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected error to be *APIError, got %T", err)
	}

	if apiErr.StatusCode != 429 {
		t.Errorf("expected status code 429, got %d", apiErr.StatusCode)
	}
	if apiErr.ErrorType != "insufficient_quota" {
		t.Errorf("expected error type 'insufficient_quota', got %q", apiErr.ErrorType)
	}
	if apiErr.Code != "insufficient_quota" {
		t.Errorf("expected code 'insufficient_quota', got %q", apiErr.Code)
	}
	if apiErr.Message != "You exceeded your current quota, please check your plan and billing details." {
		t.Errorf("unexpected message: %q", apiErr.Message)
	}
}

func TestNewAPIError_Anthropic(t *testing.T) {
	rawJSON := `{
		"type": "error",
		"error": {
			"type": "rate_limit_error",
			"message": "Number of request tokens has exceeded your per-minute limit."
		}
	}`

	err := NewAPIError(429, []byte(rawJSON))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected error to be *APIError, got %T", err)
	}

	if apiErr.StatusCode != 429 {
		t.Errorf("expected status code 429, got %d", apiErr.StatusCode)
	}
	if apiErr.ErrorType != "rate_limit_error" {
		t.Errorf("expected error type 'rate_limit_error', got %q", apiErr.ErrorType)
	}
	if apiErr.Message != "Number of request tokens has exceeded your per-minute limit." {
		t.Errorf("unexpected message: %q", apiErr.Message)
	}
}

func TestNewAPIError_Fallback(t *testing.T) {
	rawText := "some non-json error message"

	err := NewAPIError(500, []byte(rawText))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected error to be *APIError, got %T", err)
	}

	if apiErr.StatusCode != 500 {
		t.Errorf("expected status code 500, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != rawText {
		t.Errorf("expected message %q, got %q", rawText, apiErr.Message)
	}
}

func TestNewAPIError_FallbackJSON(t *testing.T) {
	unrecognizedJSON := `{"foo":"bar","nested":{"num":123}}`

	err := NewAPIError(400, []byte(unrecognizedJSON))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected error to be *APIError, got %T", err)
	}

	if apiErr.StatusCode != 400 {
		t.Errorf("expected status code 400, got %d", apiErr.StatusCode)
	}

	expectedPretty := "{\n  \"foo\": \"bar\",\n  \"nested\": {\n    \"num\": 123\n  }\n}"
	if apiErr.Message != expectedPretty {
		t.Errorf("expected pretty JSON:\n%s\ngot:\n%s", expectedPretty, apiErr.Message)
	}
}

func TestAPIError_Error(t *testing.T) {
	apiErr := &APIError{
		StatusCode: 429,
		Message:    "Quota Exceeded",
	}

	expected := "API request failed with status 429: Quota Exceeded"
	if apiErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, apiErr.Error())
	}

	emptyMsgErr := &APIError{
		StatusCode: 500,
	}
	expectedEmpty := "API request failed with status 500"
	if emptyMsgErr.Error() != expectedEmpty {
		t.Errorf("expected %q, got %q", expectedEmpty, emptyMsgErr.Error())
	}
}

func TestSendWithRetry_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	body, err := SendWithRetry(context.Background(), server.Client(), req, 3)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if string(body) != `{"ok": true}` {
		t.Errorf("expected body %q, got %q", `{"ok": true}`, string(body))
	}
}

func TestSendWithRetry_RetryOn429(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	body, err := SendWithRetry(context.Background(), server.Client(), req, 5)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if string(body) != `{"ok": true}` {
		t.Errorf("expected body %q, got %q", `{"ok": true}`, string(body))
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts (2 retries + success), got %d", attempts)
	}
}

func TestSendWithRetry_RetryOn500(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	body, err := SendWithRetry(context.Background(), server.Client(), req, 5)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if string(body) != `{"ok": true}` {
		t.Errorf("expected body %q, got %q", `{"ok": true}`, string(body))
	}
}

func TestSendWithRetry_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"unavailable"}`))
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, err := SendWithRetry(context.Background(), server.Client(), req, 2)
	if err == nil {
		t.Fatal("expected error after max retries exceeded, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", apiErr.StatusCode)
	}
}

func TestSendWithRetry_4xxNoRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, err := SendWithRetry(context.Background(), server.Client(), req, 3)
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
}

func TestSendWithRetry_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, err := SendWithRetry(ctx, server.Client(), req, 3)
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}

func TestWaitBackoff(t *testing.T) {
	start := time.Now()
	waitBackoff(context.Background(), 0) // 1s
	elapsed := time.Since(start)
	if elapsed < 500*time.Millisecond {
		t.Errorf("expected backoff ~1s, got %v", elapsed)
	}
}

func TestWaitBackoff_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return immediately without blocking
	start := time.Now()
	waitBackoff(ctx, 3) // would be 8s if not cancelled
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected fast return on cancelled context, took %v", elapsed)
	}
}

func TestSendWithRetry_NetworkError(t *testing.T) {
	// Create a server that immediately closes connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	server.Close() // Close before use to cause connection refused

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, err := SendWithRetry(context.Background(), &http.Client{}, req, 1)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}
