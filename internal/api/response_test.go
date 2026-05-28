package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_WritesContentType(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestJSON_WritesStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusCreated, map[string]string{"id": "123"})

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestJSON_WritesBody(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "hello"}
	JSON(w, http.StatusOK, data)

	var decoded map[string]string
	err := json.NewDecoder(w.Body).Decode(&decoded)
	require.NoError(t, err)
	assert.Equal(t, "hello", decoded["message"])
}

func TestJSON_ArrayResponse(t *testing.T) {
	w := httptest.NewRecorder()
	items := []string{"a", "b", "c"}
	JSON(w, http.StatusOK, items)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusOK, w.Code)

	var decoded []string
	err := json.NewDecoder(w.Body).Decode(&decoded)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, decoded)
}

func TestPaginatedResponse_HasItemsAndPagination(t *testing.T) {
	items := []string{"x", "y", "z"}
	params := PaginationParams{Limit: 10, Offset: 0}
	resp := NewPaginatedResponse(items, 42, params)

	assert.Equal(t, []string{"x", "y", "z"}, resp.Items)
	assert.Equal(t, 10, resp.Pagination.Limit)
	assert.Equal(t, 0, resp.Pagination.Offset)
	assert.Equal(t, 42, resp.Pagination.Total)
}

func TestPaginatedResponse_EmptyItems(t *testing.T) {
	items := []int{}
	params := PaginationParams{Limit: 100, Offset: 0}
	resp := NewPaginatedResponse(items, 0, params)

	assert.Empty(t, resp.Items)
	assert.Equal(t, 100, resp.Pagination.Limit)
	assert.Equal(t, 0, resp.Pagination.Total)
}

func TestPaginatedResponse_JSONStructure(t *testing.T) {
	type widget struct {
		ID int `json:"id"`
	}

	items := []widget{{ID: 1}, {ID: 2}}
	params := PaginationParams{Limit: 50, Offset: 25}
	resp := NewPaginatedResponse(items, 2, params)

	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, resp)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	// Verify items array
	itemsRaw, ok := body["items"].([]interface{})
	require.True(t, ok, "response must have 'items' key")
	assert.Len(t, itemsRaw, 2)

	// Verify pagination object
	paginationRaw, ok := body["pagination"].(map[string]interface{})
	require.True(t, ok, "response must have 'pagination' key")
	assert.Equal(t, float64(50), paginationRaw["limit"])
	assert.Equal(t, float64(25), paginationRaw["offset"])
	assert.Equal(t, float64(2), paginationRaw["total"])
}
