package db

import (
	"context"
	"errors"
	"math"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// paginationRecord is a test-only persistent model used to verify filtered pagination.
type paginationRecord struct {
	// ID identifies the record and defines the test's stable result order.
	ID uint `gorm:"primaryKey"`
	// Name stores the value returned in pagination assertions.
	Name string
	// Active controls whether the record belongs to the filtered result set.
	Active bool
}

// TestPaginate_returnsFilteredPage verifies count metadata and records use the same filtered query.
func TestPaginate_returnsFilteredPage(t *testing.T) {
	gdb := openPaginationTestDB(t)
	records := []paginationRecord{
		{Name: "one", Active: true},
		{Name: "two", Active: false},
		{Name: "three", Active: true},
		{Name: "four", Active: true},
		{Name: "five", Active: true},
	}
	if err := gdb.Create(&records).Error; err != nil {
		t.Fatalf("create pagination records: %v", err)
	}

	query := gdb.Where("active = ?", true).Order("id ASC").Limit(1).Offset(1)
	got, err := Paginate[paginationRecord](context.Background(), query, Pagination{
		Page:     2,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("Paginate: %v", err)
	}
	if got.Page != 2 || got.PageSize != 2 || got.TotalItems != 4 || got.TotalPages != 2 {
		t.Fatalf("metadata = %+v, want page=2 page_size=2 total_items=4 total_pages=2", got)
	}
	if len(got.Items) != 2 || got.Items[0].Name != "four" || got.Items[1].Name != "five" {
		t.Fatalf("items = %+v, want records four and five", got.Items)
	}
}

// TestPaginate_normalizesBounds verifies defaults, maximum size, and out-of-range pages.
func TestPaginate_normalizesBounds(t *testing.T) {
	gdb := openPaginationTestDB(t)
	if err := gdb.Create(&paginationRecord{Name: "only", Active: true}).Error; err != nil {
		t.Fatalf("create pagination record: %v", err)
	}

	first, err := Paginate[paginationRecord](context.Background(), gdb, Pagination{})
	if err != nil {
		t.Fatalf("Paginate defaults: %v", err)
	}
	if first.Page != firstPage || first.PageSize != DefaultPageSize || len(first.Items) != 1 {
		t.Fatalf("default page = %+v, want page=1 page_size=%d with one item", first, DefaultPageSize)
	}

	outside, err := Paginate[paginationRecord](context.Background(), gdb, Pagination{
		Page:     2,
		PageSize: MaxPageSize + 1,
	})
	if err != nil {
		t.Fatalf("Paginate outside result set: %v", err)
	}
	if outside.PageSize != MaxPageSize || outside.TotalItems != 1 || outside.TotalPages != 1 {
		t.Fatalf("outside metadata = %+v, want capped size and preserved totals", outside)
	}
	if outside.Items == nil || len(outside.Items) != 0 {
		t.Fatalf("outside items = %#v, want a non-nil empty slice", outside.Items)
	}
}

// TestPaginate_supportsExplicitModel verifies projections can count an explicitly selected model.
func TestPaginate_supportsExplicitModel(t *testing.T) {
	gdb := openPaginationTestDB(t)
	if err := gdb.Create(&paginationRecord{Name: "projected", Active: true}).Error; err != nil {
		t.Fatalf("create pagination record: %v", err)
	}

	type nameProjection struct {
		Name string
	}
	got, err := Paginate[nameProjection](
		context.Background(),
		gdb.Model(&paginationRecord{}).Select("name").Order("id ASC"),
		Pagination{Page: 1, PageSize: 10},
	)
	if err != nil {
		t.Fatalf("Paginate projection: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Name != "projected" || got.TotalItems != 1 {
		t.Fatalf("projection result = %+v, want projected name and total_items=1", got)
	}
}

// TestPaginate_rejectsInvalidInputs verifies nil queries and impossible offsets return errors.
func TestPaginate_rejectsInvalidInputs(t *testing.T) {
	_, err := Paginate[paginationRecord](context.Background(), nil, Pagination{})
	if !errors.Is(err, errNilPaginationQuery) {
		t.Fatalf("nil query error = %v, want %v", err, errNilPaginationQuery)
	}

	gdb := openPaginationTestDB(t)
	_, err = Paginate[paginationRecord](context.Background(), gdb, Pagination{
		Page:     math.MaxInt,
		PageSize: MaxPageSize,
	})
	if err == nil {
		t.Fatal("overflowing offset returned nil error")
	}
}

// openPaginationTestDB opens an isolated single-connection SQLite database for a pagination test.
func openPaginationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get *sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close pagination DB: %v", err)
		}
	})
	if err := gdb.AutoMigrate(&paginationRecord{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return gdb
}
