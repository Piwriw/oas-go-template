package store_test

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/piwriw/oas-go-template/internal/db/store"

	"github.com/piwriw/oas-go-template/internal/db"
	"github.com/piwriw/oas-go-template/internal/db/models"
)

// TestNewExampleStoreRejectsNilDB verifies a store cannot be built without its database dependency.
func TestNewExampleStoreRejectsNilDB(t *testing.T) {
	_, err := store.NewExampleStore(nil)
	if !errors.Is(err, gorm.ErrInvalidDB) {
		t.Fatalf("NewExampleStore(nil) error = %v, want %v", err, gorm.ErrInvalidDB)
	}
}

// TestExampleStoreCreateGetAndList verifies creation, lookup, and stable pagination.
func TestExampleStoreCreateGetAndList(t *testing.T) {
	exampleStore := openExampleStore(t)
	ctx := context.Background()

	examples := []*models.Example{
		{Name: "one"},
		{Name: "two"},
		{Name: "three"},
	}
	for _, example := range examples {
		if err := exampleStore.Create(ctx, example); err != nil {
			t.Fatalf("Create(%q): %v", example.Name, err)
		}
		if example.ID == 0 || example.CreatedAt.IsZero() || example.UpdatedAt.IsZero() {
			t.Fatalf("created example = %+v, want generated ID and timestamps", example)
		}
	}

	got, err := exampleStore.GetByID(ctx, examples[1].ID)
	if err != nil {
		t.Fatalf("GetByID(%d): %v", examples[1].ID, err)
	}
	if got.Name != "two" {
		t.Fatalf("GetByID(%d).Name = %q, want %q", examples[1].ID, got.Name, "two")
	}

	page, err := exampleStore.List(ctx, db.Pagination{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalItems != 3 || page.TotalPages != 2 || len(page.Items) != 1 {
		t.Fatalf("List result = %+v, want three items across two pages", page)
	}
	if page.Items[0].ID != examples[2].ID {
		t.Fatalf("second page ID = %d, want %d", page.Items[0].ID, examples[2].ID)
	}
}

// TestExampleStoreUpdateAndDelete verifies mutable fields and missing-record errors.
func TestExampleStoreUpdateAndDelete(t *testing.T) {
	exampleStore := openExampleStore(t)
	ctx := context.Background()
	example := &models.Example{Name: "original"}
	if err := exampleStore.Create(ctx, example); err != nil {
		t.Fatalf("Create: %v", err)
	}

	example.Name = "updated"
	if err := exampleStore.Update(ctx, example); err != nil {
		t.Fatalf("Update(%d): %v", example.ID, err)
	}
	updated, err := exampleStore.GetByID(ctx, example.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if updated.Name != "updated" {
		t.Fatalf("updated Name = %q, want %q", updated.Name, "updated")
	}

	if err := exampleStore.Delete(ctx, example.ID); err != nil {
		t.Fatalf("Delete(%d): %v", example.ID, err)
	}
	if _, err := exampleStore.GetByID(ctx, example.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetByID after delete error = %v, want %v", err, store.ErrNotFound)
	}
	if err := exampleStore.Update(ctx, example); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Update deleted example error = %v, want %v", err, store.ErrNotFound)
	}
	if err := exampleStore.Delete(ctx, example.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Delete deleted example error = %v, want %v", err, store.ErrNotFound)
	}
}

// TestExampleStoreRejectsNilModels verifies model-taking operations reject nil pointers.
func TestExampleStoreRejectsNilModels(t *testing.T) {
	exampleStore := openExampleStore(t)
	ctx := context.Background()

	if err := exampleStore.Create(ctx, nil); !errors.Is(err, gorm.ErrInvalidValue) {
		t.Fatalf("Create(nil) error = %v, want %v", err, gorm.ErrInvalidValue)
	}
	if err := exampleStore.Update(ctx, nil); !errors.Is(err, gorm.ErrInvalidValue) {
		t.Fatalf("Update(nil) error = %v, want %v", err, gorm.ErrInvalidValue)
	}
}

// openExampleStore opens an isolated SQLite database and prepares the example table.
func openExampleStore(t *testing.T) *store.ExampleStore {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
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
	sqlDB.SetMaxIdleConns(1)
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close example store DB: %v", err)
		}
	})
	if err := gdb.AutoMigrate(&models.Example{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	exampleStore, err := store.NewExampleStore(gdb)
	if err != nil {
		t.Fatalf("NewExampleStore: %v", err)
	}
	return exampleStore
}
