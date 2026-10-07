package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/piwriw/oas-go-template/internal/db"
)

// TimeStore retrieves timestamps from the database clock.
type TimeStore struct {
	db *gorm.DB
}

// NewTimeStore constructs a time store from an initialized database connection.
func NewTimeStore(gdb *gorm.DB) (*TimeStore, error) {
	if gdb == nil {
		return nil, fmt.Errorf("create time store: %w", gorm.ErrInvalidDB)
	}
	return &TimeStore{db: gdb}, nil
}

// GetCurrentTime returns the database's current timestamp using the caller's context.
func (s *TimeStore) GetCurrentTime(ctx context.Context) (time.Time, error) {
	var query string
	switch s.db.Name() {
	case db.DriverPostgres, db.DriverMySQL:
		query = "SELECT NOW()"
	case db.DriverSQLite:
		query = "SELECT CURRENT_TIMESTAMP"
	default:
		return time.Time{}, fmt.Errorf("get database time: unsupported database dialect %q", s.db.Name())
	}

	row := s.db.WithContext(ctx).Raw(query).Row()
	if s.db.Name() == db.DriverSQLite {
		var value string
		if err := row.Scan(&value); err != nil {
			return time.Time{}, fmt.Errorf("get database time: %w", err)
		}
		currentTime, err := time.Parse(time.DateTime, value)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse database time: %w", err)
		}
		return currentTime, nil
	}

	var currentTime time.Time
	if err := row.Scan(&currentTime); err != nil {
		return time.Time{}, fmt.Errorf("get database time: %w", err)
	}
	return currentTime, nil
}
