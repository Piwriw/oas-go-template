package db

import (
	"context"
	"errors"
	"fmt"
	"math"

	"gorm.io/gorm"
)

const (
	// DefaultPageSize is used when the requested page size is not positive.
	DefaultPageSize = 10
	// MaxPageSize caps a single page to keep accidental queries bounded.
	MaxPageSize = 10000
	firstPage   = 1
)

var errNilPaginationQuery = errors.New("paginate: nil gorm query")

// Pagination describes a one-based page request.
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PageResult contains one page of records and its navigation metadata.
type PageResult[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int64 `json:"total_pages"`
}

// Paginate counts and loads one page from a filtered Gorm query.
func Paginate[T any](ctx context.Context, query *gorm.DB, pagination Pagination) (PageResult[T], error) {
	pagination = normalizePagination(pagination)
	result := PageResult[T]{
		Items:    make([]T, 0),
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}
	if query == nil {
		return result, errNilPaginationQuery
	}

	offset, err := paginationOffset(pagination)
	if err != nil {
		return result, err
	}

	baseQuery := query.WithContext(ctx).Limit(-1).Offset(-1)
	if baseQuery.Statement.Model == nil {
		baseQuery = baseQuery.Model(new(T))
	}
	if err := baseQuery.Count(&result.TotalItems).Error; err != nil {
		return result, fmt.Errorf("count paginated rows: %w", err)
	}

	result.TotalPages = result.TotalItems / int64(pagination.PageSize)
	if result.TotalItems%int64(pagination.PageSize) != 0 {
		result.TotalPages++
	}
	if result.TotalItems == 0 || int64(offset) >= result.TotalItems {
		return result, nil
	}

	result.Items = make([]T, 0, pagination.PageSize)
	if err := baseQuery.Offset(offset).Limit(pagination.PageSize).Find(&result.Items).Error; err != nil {
		return result, fmt.Errorf("load paginated rows: %w", err)
	}
	return result, nil
}

// normalizePagination applies one-based defaults and the configured page-size cap.
func normalizePagination(pagination Pagination) Pagination {
	if pagination.Page < firstPage {
		pagination.Page = firstPage
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = DefaultPageSize
	}
	if pagination.PageSize > MaxPageSize {
		pagination.PageSize = MaxPageSize
	}
	return pagination
}

// paginationOffset calculates a Gorm-compatible offset without integer overflow.
func paginationOffset(pagination Pagination) (int, error) {
	if pagination.Page-firstPage > math.MaxInt/pagination.PageSize {
		return 0, fmt.Errorf("pagination offset overflows int: page=%d page_size=%d", pagination.Page, pagination.PageSize)
	}
	return (pagination.Page - firstPage) * pagination.PageSize, nil
}
