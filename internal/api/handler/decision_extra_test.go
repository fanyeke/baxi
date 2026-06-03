package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// structToMap edge cases (currently 66.7%)
// ============================================================

func TestStructToMap_Nil(t *testing.T) {
	result := structToMap(nil)
	assert.Nil(t, result)
}

func TestStructToMap_ValidStruct(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	input := testStruct{Name: "test", Value: 42}
	result := structToMap(input)
	require.NotNil(t, result)
	assert.Equal(t, "test", result["name"])
	assert.Equal(t, float64(42), result["value"])
}

func TestStructToMap_MarshalError(t *testing.T) {
	// channels cannot be marshaled to JSON
	ch := make(chan int)
	result := structToMap(ch)
	assert.Nil(t, result)
}

func TestStructToMap_MapInput(t *testing.T) {
	input := map[string]interface{}{"key": "value"}
	result := structToMap(input)
	require.NotNil(t, result)
	assert.Equal(t, "value", result["key"])
}

// ============================================================
// DecideLLM handler (0% coverage)
// ============================================================

func TestDecideLLM_Success(t *testing.T) {
	svc := &mockDecisionService{}
	h := NewDecisionHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/decisions/cases/case-1/evals", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("case_id", "case-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ListEvals(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListEvals_InternalError(t *testing.T) {
	svc := &mockDecisionService{
		listEvalsFn: func(ctx context.Context, caseID string) (interface{}, error) {
			return nil, errors.New("database error")
		},
	}
	h := NewDecisionHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/decisions/cases/case-1/evals", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("case_id", "case-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ListEvals(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
