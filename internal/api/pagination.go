package api

import (
	"net/http"

	"baxi/internal/httputil"
)

type PaginationParams = httputil.PaginationParams
type PaginationMeta = httputil.PaginationMeta
type SortOption = httputil.SortOption

func ParsePagination(r *http.Request) (httputil.PaginationParams, error) {
	return httputil.ParsePagination(r)
}

func ParseSort(r *http.Request, allowedSorts map[string]string) (string, error) {
	return httputil.ParseSort(r, allowedSorts)
}
