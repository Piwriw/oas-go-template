package db

import (
	"context"
	"fmt"
	"testing"
	"testing/fstest"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	migrationTestVersion    = "20260801143000"
	migrationTestTable      = "migration_test_widgets"
	migrationTestCreateFile = "migrations/" + migrationTestVersion + "_create_widgets.up.sql"
	migrationTestDropFile   = "migrations/" + migrationTestVersion + "_create_widgets.down.sql"
)

// TestRunMigrations_appliesAndSkipsRecordedVersion verifies applied schema versions are recorded and not replayed.
func TestRunMigrations_appliesAndSkipsRecordedVersion(t *testing.T) {
	cfg := migrationTestConfig(t)
	gdb := openMigrationTestDB(t, cfg.DSN)
	migrationFS := validMigrationFS(
		"CREATE TABLE "+migrationTestTable+" (id INTEGER PRIMARY KEY, name TEXT NOT NULL);",
		"DROP TABLE "+migrationTestTable+";",
	)

	if err := runMigrations(context.Background(), gdb, cfg, migrationFS); err != nil {
		t.Fatalf("first runMigrations: %v", err)
	}
	if !gdb.Migrator().HasTable(migrationTestTable) {
		t.Fatal("first runMigrations did not create the SQL migration table")
	}

	var record struct {
		Version uint64
		Dirty   bool
	}
	if err := gdb.Table(migrationTableName).Take(&record).Error; err != nil {
		t.Fatalf("read migration record: %v", err)
	}
	if got, want := fmt.Sprint(record.Version), migrationTestVersion; got != want {
		t.Errorf("recorded version = %q, want %q", got, want)
	}
	if record.Dirty {
		t.Error("successful migration was recorded as dirty")
	}

	// Removing the table makes a repeated SQL execution observable. Because
	// the version is already recorded, the second run must leave it absent.
	if err := gdb.Migrator().DropTable(migrationTestTable); err != nil {
		t.Fatalf("drop migrated table: %v", err)
	}
	if err := runMigrations(context.Background(), gdb, cfg, migrationFS); err != nil {
		t.Fatalf("second runMigrations: %v", err)
	}
	if gdb.Migrator().HasTable(migrationTestTable) {
		t.Error("recorded migration ran a second time")
	}
}

// TestRunMigrations_failedSQLLeavesDirtyVersion verifies failed schema changes block startup with a dirty version.
func TestRunMigrations_failedSQLLeavesDirtyVersion(t *testing.T) {
	cfg := migrationTestConfig(t)
	gdb := openMigrationTestDB(t, cfg.DSN)
	migrationFS := validMigrationFS("THIS IS NOT SQL;", "SELECT 1;")

	if err := runMigrations(context.Background(), gdb, cfg, migrationFS); err == nil {
		t.Fatal("runMigrations with invalid SQL succeeded, want error")
	}

	var record struct {
		Version uint64
		Dirty   bool
	}
	if err := gdb.Table(migrationTableName).Take(&record).Error; err != nil {
		t.Fatalf("read dirty migration record: %v", err)
	}
	if !record.Dirty {
		t.Error("failed migration was not marked dirty")
	}
	if err := runMigrations(context.Background(), gdb, cfg, migrationFS); err == nil {
		t.Fatal("runMigrations accepted a dirty database, want error")
	}
}

// TestValidateMigrationFiles verifies embedded schema versions require paired up and down scripts.
func TestValidateMigrationFiles(t *testing.T) {
	tests := []struct {
		name        string
		migrationFS fstest.MapFS
	}{
		{
			name: "invalid filename",
			migrationFS: fstest.MapFS{
				"migrations/create_widgets.sql": {Data: []byte("SELECT 1;")},
			},
		},
		{
			name: "invalid timestamp",
			migrationFS: fstest.MapFS{
				"migrations/20261301143000_create_widgets.up.sql":   {Data: []byte("SELECT 1;")},
				"migrations/20261301143000_create_widgets.down.sql": {Data: []byte("SELECT 1;")},
			},
		},
		{
			name: "missing down",
			migrationFS: fstest.MapFS{
				migrationTestCreateFile: {Data: []byte("SELECT 1;")},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := validateMigrationFiles(test.migrationFS); err == nil {
				t.Fatal("validateMigrationFiles succeeded, want error")
			}
		})
	}
}

// TestMigrate_disabledDBIsNoop verifies schema migration is skipped when database support is disabled.
func TestMigrate_disabledDBIsNoop(t *testing.T) {
	if err := Migrate(context.Background(), nil, Config{}); err != nil {
		t.Fatalf("Migrate disabled DB: %v", err)
	}
}

// validMigrationFS builds one paired in-memory migration version for database tests.
func validMigrationFS(upSQL, downSQL string) fstest.MapFS {
	return fstest.MapFS{
		migrationTestCreateFile: {Data: []byte(upSQL)},
		migrationTestDropFile:   {Data: []byte(downSQL)},
	}
}

// migrationTestConfig returns an isolated SQLite configuration for one migration test.
func migrationTestConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		Driver: "sqlite",
		DSN:    t.TempDir() + "/migration.db",
	}
}

// openMigrationTestDB opens a disposable SQLite connection for migration assertions.
func openMigrationTestDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite test DB: %v", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get test *sql.DB: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test DB: %v", err)
		}
	})
	return gdb
}
