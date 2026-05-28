package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePagination_Defaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	params, err := ParsePagination(r)

	require.NoError(t, err)
	assert.Equal(t, 100, params.Limit)
	assert.Equal(t, 0, params.Offset)
}

func TestParsePagination_CustomValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?limit=50&offset=10", nil)
	params, err := ParsePagination(r)

	require.NoError(t, err)
	assert.Equal(t, 50, params.Limit)
	assert.Equal(t, 10, params.Offset)
}

func TestParsePagination_InvalidLimit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?limit=abc", nil)
	_, err := ParsePagination(r)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid limit")
}

func TestParsePagination_InvalidOffset(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?offset=abc", nil)
	_, err := ParsePagination(r)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid offset")
}

func TestParsePagination_BoundsChecking(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedLimit  int
		expectedOffset int
	}{
		{"minimum limit", "?limit=0", 1, 0},
		{"negative limit", "?limit=-5", 1, 0},
		{"maximum limit", "?limit=2000", 1000, 0},
		{"valid offset", "?offset=10", 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/test"+tt.query, nil)
			params, err := ParsePagination(r)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLimit, params.Limit)
			assert.Equal(t, tt.expectedOffset, params.Offset)
		})
	}
}

func TestParseSort_DefaultSort(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	allowedSorts := map[string]string{"name": "name", "created_at": "created_at"}

	sort, err := ParseSort(r, allowedSorts)

	require.NoError(t, err)
	assert.Equal(t, "created_at DESC", sort)
}

func TestParseSort_ValidSort(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?sort=name%20ASC", nil)
	allowedSorts := map[string]string{"name": "name", "created_at": "created_at"}

	sort, err := ParseSort(r, allowedSorts)

	require.NoError(t, err)
	assert.Equal(t, "name ASC", sort)
}

func TestParseSort_InvalidField(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?sort=invalid_field%20ASC", nil)
	allowedSorts := map[string]string{"name": "name", "created_at": "created_at"}

	_, err := ParseSort(r, allowedSorts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sort field")
}

func TestParseSort_InvalidOrder(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?sort=name%20INVALID", nil)
	allowedSorts := map[string]string{"name": "name", "created_at": "created_at"}

	_, err := ParseSort(r, allowedSorts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sort order")
}

func TestJSON_SuccessResponse(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	JSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}
