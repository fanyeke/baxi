package httputil

import (
	"encoding/json"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

type PaginatedResponse[T any] struct {
	Items      []T            `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

func NewPaginatedResponse[T any](items []T, total int, pagination PaginationParams) PaginatedResponse[T] {
	return PaginatedResponse[T]{
		Items: items,
		Pagination: PaginationMeta{
			Limit:  pagination.Limit,
			Offset: pagination.Offset,
			Total:  total,
		},
	}
}
