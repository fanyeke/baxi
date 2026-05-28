package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// apiBaseURL returns the base URL for the baxi API.
// Uses BAXI_API_BASE_URL env var, defaults to http://localhost:8080.
func apiBaseURL() string {
	if base := os.Getenv("BAXI_API_BASE_URL"); base != "" {
		return strings.TrimRight(base, "/")
	}
	return "http://localhost:8080"
}

// apiToken returns the bearer token for API authentication from API_BEARER_TOKEN env var.
func apiToken() string {
	return os.Getenv("API_BEARER_TOKEN")
}

// apiURL constructs a full API URL from a path. Path should start with "/".
func apiURL(path string) string {
	return apiBaseURL() + path
}

// newHTTPClient creates an HTTP client with a 30-second timeout.
func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// setAuth sets the Authorization Bearer header on a request if a token is configured.
func setAuth(req *http.Request) {
	if token := apiToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// drainBody reads and closes the response body so the connection can be reused.
func drainBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// checkResponse returns an error for non-2xx HTTP responses, including the
// status code and up to 512 bytes of the response body. For 2xx responses
// the body is NOT drained and must be closed by the caller.
func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	return fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

// apiGet performs an authenticated GET request to the API.
// The response body must be closed by the caller.
func apiGet(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", apiURL(path), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	setAuth(req)
	return newHTTPClient().Do(req)
}

// apiPost performs an authenticated POST request to the API with an optional
// JSON body. Pass an empty string for requests with no body.
// The response body must be closed by the caller.
func apiPost(path, body string) (*http.Response, error) {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest("POST", apiURL(path), r)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}
	setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	return newHTTPClient().Do(req)
}
