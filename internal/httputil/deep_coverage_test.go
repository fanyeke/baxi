package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── JSON ──────────────────────────────────────────────────────────────

func TestJSON_ErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bad request")
}

func TestJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ──── NewPaginatedResponse ──────────────────────────────────────────────

func TestNewPaginatedResponse_WithItems(t *testing.T) {
	items := []string{"a", "b", "c"}
	total := 10
	params := PaginationParams{Limit: 3, Offset: 0}

	result := NewPaginatedResponse(items, total, params)
	assert.Equal(t, 3, len(result.Items))
	assert.Equal(t, "a", result.Items[0])
	assert.Equal(t, 3, result.Pagination.Limit)
	assert.Equal(t, 0, result.Pagination.Offset)
	assert.Equal(t, 10, result.Pagination.Total)
}

func TestNewPaginatedResponse_EmptyItems(t *testing.T) {
	items := []int{}
	params := PaginationParams{Limit: 100, Offset: 50}

	result := NewPaginatedResponse(items, 0, params)
	assert.Empty(t, result.Items)
	assert.Equal(t, 100, result.Pagination.Limit)
	assert.Equal(t, 50, result.Pagination.Offset)
	assert.Equal(t, 0, result.Pagination.Total)
}

func TestNewPaginatedResponse_NilItems(t *testing.T) {
	params := PaginationParams{Limit: 10, Offset: 0}
	result := NewPaginatedResponse[any](nil, 0, params)
	assert.Empty(t, result.Items)
}

func TestNewPaginatedResponse_JSONSerialization(t *testing.T) {
	items := []string{"x", "y"}
	params := PaginationParams{Limit: 2, Offset: 5}

	result := NewPaginatedResponse(items, 100, params)

	data, err := json.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"items"`)
	assert.Contains(t, string(data), `"pagination"`)
}

// ──── ParsePagination additional edge cases ─────────────────────────────

func TestParsePagination_LimitExactly1(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/test?limit=1", nil)
	params, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 1, params.Limit)
}

func TestParsePagination_LimitExactly1000(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/test?limit=1000", nil)
	params, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 1000, params.Limit)
}

func TestParsePagination_OnlyOffset(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/test?offset=25", nil)
	params, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 100, params.Limit)
	assert.Equal(t, 25, params.Offset)
}

// ──── ParseSort additional edge cases ──────────────────────────────────

func TestParseSort_ASCExplicit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/test?sort=name+ASC", nil)
	allowedSorts := map[string]string{"name": "name"}

	sort, err := ParseSort(r, allowedSorts)
	require.NoError(t, err)
	assert.Equal(t, "name ASC", sort)
}

func TestParseSort_DESCExplicit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/test?sort=name+DESC", nil)
	allowedSorts := map[string]string{"name": "name"}

	sort, err := ParseSort(r, allowedSorts)
	require.NoError(t, err)
	assert.Equal(t, "name DESC", sort)
}

func TestParseSort_EmptyString(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/test?sort=", nil)
	allowedSorts := map[string]string{"name": "name"}

	sort, err := ParseSort(r, allowedSorts)
	require.NoError(t, err)
	assert.Equal(t, "created_at DESC", sort)
}

func TestParseSort_CaseInsensitiveOrder(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/test?sort=name+desc", nil)
	allowedSorts := map[string]string{"name": "name"}

	sort, err := ParseSort(r, allowedSorts)
	require.NoError(t, err)
	assert.Equal(t, "name DESC", sort)
}

// ──── ParseSort with multiple allowed sorts ─────────────────────────────

func TestParseSort_MultipleAllowedSorts(t *testing.T) {
	allowedSorts := map[string]string{
		"created_at": "created_at",
		"severity":   "severity",
		"name":       "name",
	}

	r := httptest.NewRequest(http.MethodGet, "/api/test?sort=severity+DESC", nil)
	sort, err := ParseSort(r, allowedSorts)
	require.NoError(t, err)
	assert.Equal(t, "severity DESC", sort)
}

// ──── SortOption ────────────────────────────────────────────────────────

func TestSortOption_Fields(t *testing.T) {
	s := SortOption{Field: "name", Order: "ASC"}
	assert.Equal(t, "name", s.Field)
	assert.Equal(t, "ASC", s.Order)
}
