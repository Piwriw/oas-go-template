// Package store provides database operations for persistent application models.
package store

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/piwriw/oas-go-template/internal/db"
	"github.com/piwriw/oas-go-template/internal/db/models"
)

// ExampleStore persists and retrieves examples through Gorm.
type ExampleStore struct {
	db *gorm.DB
}

// NewExampleStore constructs an example store from an initialized database connection.
func NewExampleStore(gdb *gorm.DB) (*ExampleStore, error) {
	if gdb == nil {
		return nil, fmt.Errorf("create example store: %w", gorm.ErrInvalidDB)
	}
	return &ExampleStore{db: gdb}, nil
}

// Create inserts an example and populates its generated ID and timestamps.
func (s *ExampleStore) Create(ctx context.Context, example *models.Example) error {
	if example == nil {
		return fmt.Errorf("create example: %w", gorm.ErrInvalidValue)
	}
	if err := s.db.WithContext(ctx).Create(example).Error; err != nil {
		return fmt.Errorf("create example: %w", err)
	}
	return nil
}

// GetByID returns the example identified by its primary key.
func (s *ExampleStore) GetByID(ctx context.Context, id uint64) (*models.Example, error) {
	var example models.Example
	if err := s.db.WithContext(ctx).First(&example, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get example %d: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("get example %d: %w", id, err)
	}
	return &example, nil
}

// List returns examples in stable primary-key order with pagination metadata.
func (s *ExampleStore) List(ctx context.Context, pagination db.Pagination) (db.PageResult[models.Example], error) {
	result, err := db.Paginate[models.Example](
		ctx,
		s.db.Model(&models.Example{}).Order("id ASC"),
		pagination,
	)
	if err != nil {
		return result, fmt.Errorf("list examples: %w", err)
	}
	return result, nil
}

// Update persists the mutable fields of an existing example.
func (s *ExampleStore) Update(ctx context.Context, example *models.Example) error {
	if example == nil {
		return fmt.Errorf("update example: %w", gorm.ErrInvalidValue)
	}

	if err := s.db.WithContext(ctx).
		Model(&models.Example{}).
		Where("id = ?", example.ID).
		Update("name", example.Name).Error; err != nil {
		return fmt.Errorf("update example %d: %w", example.ID, err)
	}
	return nil
}

// Delete removes the example identified by its primary key.
func (s *ExampleStore) Delete(ctx context.Context, id uint64) error {
	if err := s.db.WithContext(ctx).Delete(&models.Example{}, id).Error; err != nil {
		return fmt.Errorf("delete example %d: %w", id, err)
	}
	return nil
}
