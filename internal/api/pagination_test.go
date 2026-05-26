package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePagination_Defaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	p, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 100, p.Limit)
	assert.Equal(t, 0, p.Offset)
}

func TestParsePagination_LimitZero(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=0", nil)
	p, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 1, p.Limit)
}

func TestParsePagination_LimitNegative(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=-1", nil)
	p, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 1, p.Limit)
}

func TestParsePagination_LimitOverMax(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=9999", nil)
	p, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 1000, p.Limit)
}

func TestParsePagination_OffsetZero(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?offset=0", nil)
	p, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 0, p.Offset)
}

func TestParsePagination_OffsetNegative(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?offset=-1", nil)
	p, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 0, p.Offset)
}

func TestParsePagination_NonIntegerLimit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=abc", nil)
	_, err := ParsePagination(r)
	assert.Error(t, err)
}

func TestParsePagination_NonIntegerOffset(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?offset=xyz", nil)
	_, err := ParsePagination(r)
	assert.Error(t, err)
}

func TestParsePagination_CustomValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=50&offset=10", nil)
	p, err := ParsePagination(r)
	require.NoError(t, err)
	assert.Equal(t, 50, p.Limit)
	assert.Equal(t, 10, p.Offset)
}

func TestParseSort_Default(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	allowed := map[string]string{"created_at": "DESC", "updated_at": "DESC"}
	sort, err := ParseSort(r, allowed)
	require.NoError(t, err)
	assert.Equal(t, "created_at DESC", sort)
}

func TestParseSort_ValidField(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?sort=updated_at", nil)
	allowed := map[string]string{"created_at": "DESC", "updated_at": "DESC"}
	sort, err := ParseSort(r, allowed)
	require.NoError(t, err)
	assert.Equal(t, "updated_at ASC", sort)
}

func TestParseSort_ValidFieldWithOrder(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?sort=updated_at+DESC", nil)
	allowed := map[string]string{"created_at": "DESC", "updated_at": "DESC"}
	sort, err := ParseSort(r, allowed)
	require.NoError(t, err)
	assert.Equal(t, "updated_at DESC", sort)
}

func TestParseSort_InvalidField(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?sort=invalid_field", nil)
	allowed := map[string]string{"created_at": "DESC"}
	_, err := ParseSort(r, allowed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sort field")
}

func TestParseSort_InvalidOrder(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?sort=created_at+INVALID", nil)
	allowed := map[string]string{"created_at": "DESC"}
	_, err := ParseSort(r, allowed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sort order")
}

func TestParseSort_EmptyAllowed(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test?sort=any", nil)
	allowed := map[string]string{}
	_, err := ParseSort(r, allowed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sort field")
}
