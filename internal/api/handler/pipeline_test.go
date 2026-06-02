package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockPipelineRunner implements PipelineRunner for testing.
type mockPipelineRunner struct {
	runFn func(ctx context.Context, config string) (string, error)
}

func (m *mockPipelineRunner) Run(ctx context.Context, config string) (string, error) {
	return m.runFn(ctx, config)
}

func TestNewPipelineHandler_NonNil(t *testing.T) {
	h := NewPipelineHandler(&mockPipelineRunner{})
	assert.NotNil(t, h)
}

func TestHandleRun_InvalidJSON(t *testing.T) {
	h := NewPipelineHandler(&mockPipelineRunner{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/pipeline/run", strings.NewReader("not json"))

	h.HandleRun(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleRun_ServiceError(t *testing.T) {
	h := NewPipelineHandler(&mockPipelineRunner{
		runFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("pipeline execution failed")
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/pipeline/run",
		strings.NewReader(`{"config":"full"}`))
	r.Header.Set("Content-Type", "application/json")

	h.HandleRun(w, r)

	resp := w.Result()
	// The handler returns 500 with error details wrapped in JSON
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestHandleRun_Success(t *testing.T) {
	h := NewPipelineHandler(&mockPipelineRunner{
		runFn: func(_ context.Context, config string) (string, error) {
			return "run-001", nil
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/pipeline/run",
		strings.NewReader(`{"config":"ingest_raw"}`))
	r.Header.Set("Content-Type", "application/json")

	h.HandleRun(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, "run-001", body["run_id"])
	assert.Equal(t, "started", body["status"])
}

func TestHandleRun_DryRun(t *testing.T) {
	callCount := 0
	h := NewPipelineHandler(&mockPipelineRunner{
		runFn: func(_ context.Context, config string) (string, error) {
			callCount++
			return "run-001", nil
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/pipeline/run?dry_run=true",
		strings.NewReader(`{"config":"daily"}`))
	r.Header.Set("Content-Type", "application/json")

	h.HandleRun(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 0, callCount, "Run should not be called in dry_run mode")

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, "daily", body["pipeline_type"])
	assert.NotEmpty(t, body["command"])
}
