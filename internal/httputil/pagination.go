package httputil

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type PaginationParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PaginationMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type SortOption struct {
	Field string
	Order string
}

func ParsePagination(r *http.Request) (PaginationParams, error) {
	limit := 100
	offset := 0

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			return PaginationParams{}, fmt.Errorf("invalid limit: %w", err)
		}
		limit = l
	}

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		o, err := strconv.Atoi(offsetStr)
		if err != nil {
			return PaginationParams{}, fmt.Errorf("invalid offset: %w", err)
		}
		offset = o
	}

	if limit < 1 {
		limit = 1
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	return PaginationParams{Limit: limit, Offset: offset}, nil
}

func ParseSort(r *http.Request, allowedSorts map[string]string) (string, error) {
	sortStr := r.URL.Query().Get("sort")
	if sortStr == "" {
		return "created_at DESC", nil
	}

	parts := strings.Fields(sortStr)
	if len(parts) == 0 {
		return "created_at DESC", nil
	}

	field := parts[0]
	if _, ok := allowedSorts[field]; !ok {
		return "", fmt.Errorf("invalid sort field: %s", field)
	}

	order := "ASC"
	if len(parts) >= 2 {
		upperOrder := strings.ToUpper(parts[1])
		switch upperOrder {
		case "ASC", "DESC":
			order = upperOrder
		default:
			return "", fmt.Errorf("invalid sort order: %s (must be ASC or DESC)", parts[1])
		}
	}

	return field + " " + order, nil
}
