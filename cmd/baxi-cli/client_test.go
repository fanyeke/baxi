package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestApiBaseURL_Default(t *testing.T) {
	os.Unsetenv("BAXI_API_BASE_URL")
	got := apiBaseURL()
	if got != "http://localhost:8080" {
		t.Errorf("apiBaseURL() = %q, want %q", got, "http://localhost:8080")
	}
}

func TestApiBaseURL_EnvVar(t *testing.T) {
	os.Setenv("BAXI_API_BASE_URL", "https://api.example.com")
	defer os.Unsetenv("BAXI_API_BASE_URL")
	got := apiBaseURL()
	if got != "https://api.example.com" {
		t.Errorf("apiBaseURL() = %q, want %q", got, "https://api.example.com")
	}
}

func TestApiBaseURL_TrailingSlash(t *testing.T) {
	os.Setenv("BAXI_API_BASE_URL", "https://api.example.com/")
	defer os.Unsetenv("BAXI_API_BASE_URL")
	got := apiBaseURL()
	if got != "https://api.example.com" {
		t.Errorf("apiBaseURL() = %q, want %q", got, "https://api.example.com")
	}
}

func TestApiURL(t *testing.T) {
	os.Setenv("BAXI_API_BASE_URL", "http://test:9000")
	defer os.Unsetenv("BAXI_API_BASE_URL")
	got := apiURL("/api/v1/status")
	want := "http://test:9000/api/v1/status"
	if got != want {
		t.Errorf("apiURL() = %q, want %q", got, want)
	}
}

func TestApiToken_Default(t *testing.T) {
	os.Unsetenv("API_BEARER_TOKEN")
	if got := apiToken(); got != "" {
		t.Errorf("apiToken() = %q, want empty", got)
	}
}

func TestApiToken_EnvVar(t *testing.T) {
	os.Setenv("API_BEARER_TOKEN", "test-token-123")
	defer os.Unsetenv("API_BEARER_TOKEN")
	if got := apiToken(); got != "test-token-123" {
		t.Errorf("apiToken() = %q, want %q", got, "test-token-123")
	}
}

func TestNewHTTPClient(t *testing.T) {
	c := newHTTPClient()
	if c == nil {
		t.Fatal("newHTTPClient() returned nil")
	}
	if c.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", c.Timeout)
	}
}

func TestSetAuth_WithToken(t *testing.T) {
	os.Setenv("API_BEARER_TOKEN", "my-token")
	defer os.Unsetenv("API_BEARER_TOKEN")
	req, _ := http.NewRequest("GET", "/test", nil)
	setAuth(req)
	if got := req.Header.Get("Authorization"); got != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-token")
	}
}

func TestSetAuth_WithoutToken(t *testing.T) {
	os.Unsetenv("API_BEARER_TOKEN")
	req, _ := http.NewRequest("GET", "/test", nil)
	setAuth(req)
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestDrainBody_Nil(t *testing.T) {
	// Should not panic
	drainBody(nil)
}

func TestDrainBody_NonNil(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("hello"))
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	// Should not panic
	drainBody(resp)
}

func TestCheckResponse_2xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestCheckResponse_4xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`not found`))
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if err := checkResponse(resp); err == nil {
		t.Error("checkResponse(400) = nil, want error")
	}
}

func TestCheckResponse_5xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal error`))
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if err := checkResponse(resp); err == nil {
		t.Error("checkResponse(500) = nil, want error")
	}
}

func TestApiGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	os.Setenv("BAXI_API_BASE_URL", ts.URL)
	defer os.Unsetenv("BAXI_API_BASE_URL")

	resp, err := apiGet("/test")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestApiPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer ts.Close()

	os.Setenv("BAXI_API_BASE_URL", ts.URL)
	defer os.Unsetenv("BAXI_API_BASE_URL")

	resp, err := apiPost("/test", `{"name":"test"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
}

func TestApiPost_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	os.Setenv("BAXI_API_BASE_URL", ts.URL)
	defer os.Unsetenv("BAXI_API_BASE_URL")

	resp, err := apiPost("/test", "")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestApiGet_Error(t *testing.T) {
	os.Setenv("BAXI_API_BASE_URL", "http://127.0.0.1:1")
	defer os.Unsetenv("BAXI_API_BASE_URL")

	_, err := apiGet("/test")
	if err == nil {
		t.Error("apiGet to invalid URL = nil, want error")
	}
}
