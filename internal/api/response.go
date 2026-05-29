package api

import (
	"net/http"

	"baxi/internal/httputil"
)

type PaginatedResponse[T any] struct {
	Items      []T                     `json:"items"`
	Pagination httputil.PaginationMeta `json:"pagination"`
}

func NewPaginatedResponse[T any](items []T, total int, pagination httputil.PaginationParams) PaginatedResponse[T] {
	return PaginatedResponse[T]{
		Items: items,
		Pagination: httputil.PaginationMeta{
			Limit:  pagination.Limit,
			Offset: pagination.Offset,
			Total:  total,
		},
	}
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	httputil.JSON(w, status, data)
}
