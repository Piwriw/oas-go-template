package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratedatabase "github.com/golang-migrate/migrate/v4/database"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

// MigrationDirection selects whether schema migrations advance or roll back.
type MigrationDirection string

const (
	// MigrationUp applies every pending migration.
	MigrationUp MigrationDirection = "up"
	// MigrationDown rolls back exactly one applied migration.
	MigrationDown MigrationDirection = "down"

	migrationDirectory     = "migrations"
	migrationTableName     = "schema_migrations"
	migrationVersionLayout = "20060102150405"

	migrationDriverMySQL    = "mysql"
	migrationDriverPostgres = "pgx"
	migrationDriverSQLite   = "sqlite3"
)

//go:embed migrations
var migrationFiles embed.FS

// Migrate applies embedded schema changes in the requested direction.
func Migrate(ctx context.Context, gdb *gorm.DB, cfg Config, direction MigrationDirection) error {
	if cfg.Disabled() {
		return nil
	}
	return runMigrations(ctx, gdb, cfg, direction, migrationFiles)
}

// runMigrations validates and executes the requested embedded migration operation.
func runMigrations(ctx context.Context, gdb *gorm.DB, cfg Config, direction MigrationDirection, migrationFS fs.FS) (err error) {
	switch direction {
	case MigrationUp, MigrationDown:
	default:
		return fmt.Errorf("unsupported migration direction %q (want up|down)", direction)
	}

	hasMigrations, err := validateMigrationFiles(migrationFS)
	if err != nil {
		return err
	}
	if !hasMigrations && gdb == nil {
		return nil
	}

	sourceDriver, err := iofs.New(migrationFS, migrationDirectory)
	if err != nil {
		return fmt.Errorf("open migration files: %w", err)
	}

	databaseDriver, driverName, err := newMigrationDatabase(ctx, gdb, cfg)
	if err != nil {
		return errors.Join(err, closeMigrationSource(sourceDriver))
	}
	if !hasMigrations {
		return errors.Join(closeMigrationSource(sourceDriver), closeMigrationDatabase(databaseDriver))
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, driverName, databaseDriver)
	if err != nil {
		return errors.Join(
			fmt.Errorf("create migrator: %w", err),
			closeMigrationSource(sourceDriver),
			closeMigrationDatabase(databaseDriver),
		)
	}
	defer func() {
		sourceErr, databaseErr := migrator.Close()
		err = errors.Join(err, wrapCloseError("migration source", sourceErr), wrapCloseError("migration database", databaseErr))
	}()

	stopCancellation := context.AfterFunc(ctx, func() {
		select {
		case migrator.GracefulStop <- true:
		default:
		}
	})
	defer stopCancellation()

	if err := applyDirection(migrator, direction); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("migration interrupted: %w", err)
	}
	return nil
}

// applyDirection advances or rolls back the schema through the prepared migrator.
func applyDirection(migrator *migrate.Migrate, direction MigrationDirection) error {
	switch direction {
	case MigrationUp:
		if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("apply migrations: %w", err)
		}
	case MigrationDown:
		if _, _, err := migrator.Version(); errors.Is(err, migrate.ErrNilVersion) {
			return nil
		} else if err != nil {
			return fmt.Errorf("read migration version: %w", err)
		}
		if err := migrator.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("roll back migration: %w", err)
		}
	}
	return nil
}

// newMigrationDatabase opens and adapts the configured connection for schema
// migrations. Supporting another database means adding one case and one function.
func newMigrationDatabase(ctx context.Context, gdb *gorm.DB, cfg Config) (migratedatabase.Driver, string, error) {
	switch cfg.Driver {
	case "postgres", "postgresql", "pg":
		return pgxMigrationDatabase(ctx, cfg.DSN)
	case "mysql":
		return mysqlMigrationDatabase(ctx, cfg.DSN)
	case "sqlite", "sqlite3":
		return sqliteMigrationDatabase(ctx, gdb, cfg.DSN)
	default:
		return nil, "", fmt.Errorf("unsupported db.driver %q (want postgres|mysql|sqlite)", cfg.Driver)
	}
}

// pgxMigrationDatabase dials PostgreSQL for golang-migrate through the pgx/v5 adapter.
func pgxMigrationDatabase(ctx context.Context, dsn string) (migratedatabase.Driver, string, error) {
	sqlDB, err := openMigrationDB(ctx, migrationDriverPostgres, dsn)
	if err != nil {
		return nil, "", err
	}
	driver, err := migratepgx.WithInstance(sqlDB, &migratepgx.Config{
		MigrationsTable: migrationTableName,
	})
	if err != nil {
		return nil, "", errors.Join(fmt.Errorf("initialize migration database: %w", err), sqlDB.Close())
	}
	return driver, migrationDriverPostgres, nil
}

// mysqlMigrationDatabase dials MySQL with multi-statement support enabled so one
// migration file may hold several statements.
func mysqlMigrationDatabase(ctx context.Context, dsn string) (migratedatabase.Driver, string, error) {
	mysqlConfig, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, "", fmt.Errorf("parse mysql migration DSN: %w", err)
	}
	mysqlConfig.MultiStatements = true

	sqlDB, err := openMigrationDB(ctx, migrationDriverMySQL, mysqlConfig.FormatDSN())
	if err != nil {
		return nil, "", err
	}
	driver, err := migratemysql.WithInstance(sqlDB, &migratemysql.Config{
		MigrationsTable: migrationTableName,
	})
	if err != nil {
		return nil, "", errors.Join(fmt.Errorf("initialize migration database: %w", err), sqlDB.Close())
	}
	return driver, migrationDriverMySQL, nil
}

// sqliteMigrationDatabase reuses the application connection when there is one,
// because a private connection would not see the same database — and an
// in-memory DSN would get a second, empty one. Only a standalone migrator dials
// the DSN itself, and it must not close the application's connection.
func sqliteMigrationDatabase(ctx context.Context, gdb *gorm.DB, dsn string) (migratedatabase.Driver, string, error) {
	if gdb != nil {
		sqlDB, err := gdb.DB()
		if err != nil {
			return nil, "", fmt.Errorf("get sqlite migration database: %w", err)
		}
		driver, err := migratesqlite.WithInstance(sqlDB, &migratesqlite.Config{
			MigrationsTable: migrationTableName,
		})
		if err != nil {
			return nil, "", fmt.Errorf("initialize migration database: %w", err)
		}
		return nonClosingMigrationDriver{Driver: driver}, migrationDriverSQLite, nil
	}

	sqlDB, err := openMigrationDB(ctx, migrationDriverSQLite, dsn)
	if err != nil {
		return nil, "", err
	}
	driver, err := migratesqlite.WithInstance(sqlDB, &migratesqlite.Config{
		MigrationsTable: migrationTableName,
	})
	if err != nil {
		return nil, "", errors.Join(fmt.Errorf("initialize migration database: %w", err), sqlDB.Close())
	}
	return driver, migrationDriverSQLite, nil
}

// openMigrationDB dials a migration-only connection and verifies it is reachable.
func openMigrationDB(ctx context.Context, driverName, dsn string) (*sql.DB, error) {
	sqlDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open migration database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, databasePingTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, errors.Join(fmt.Errorf("migration database ping: %w", err), sqlDB.Close())
	}
	return sqlDB, nil
}

// nonClosingMigrationDriver lets golang-migrate reuse the application's SQLite connection without closing it.
type nonClosingMigrationDriver struct {
	migratedatabase.Driver
}

// Close preserves the application-owned database connection when migrations release their driver.
func (nonClosingMigrationDriver) Close() error { return nil }

// validateMigrationFiles ensures every embedded migration version has one up and one down script.
func validateMigrationFiles(migrationFS fs.FS) (bool, error) {
	entries, err := fs.ReadDir(migrationFS, migrationDirectory)
	if err != nil {
		return false, fmt.Errorf("read migration directory: %w", err)
	}

	directions := make(map[uint]map[source.Direction]struct{})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		migration, err := source.DefaultParse(entry.Name())
		if err != nil {
			return false, fmt.Errorf("invalid migration filename %q: %w", entry.Name(), err)
		}
		version, _, _ := strings.Cut(entry.Name(), "_")
		if _, err := time.Parse(migrationVersionLayout, version); err != nil {
			return false, fmt.Errorf("invalid migration timestamp %q in %q: %w", version, entry.Name(), err)
		}

		if directions[migration.Version] == nil {
			directions[migration.Version] = make(map[source.Direction]struct{})
		}
		directions[migration.Version][migration.Direction] = struct{}{}
	}

	for version, versionDirections := range directions {
		if _, ok := versionDirections[source.Up]; !ok {
			return false, fmt.Errorf("migration %d is missing its .up.sql file", version)
		}
		if _, ok := versionDirections[source.Down]; !ok {
			return false, fmt.Errorf("migration %d is missing its .down.sql file", version)
		}
	}
	return len(directions) > 0, nil
}

// closeMigrationSource releases the embedded migration source and labels close failures.
func closeMigrationSource(driver source.Driver) error {
	return wrapCloseError("migration source", driver.Close())
}

// closeMigrationDatabase releases the migration adapter without closing the shared SQL connection.
func closeMigrationDatabase(driver migratedatabase.Driver) error {
	return wrapCloseError("migration database", driver.Close())
}

// wrapCloseError adds migration resource context to non-nil cleanup failures.
func wrapCloseError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("close %s: %w", name, err)
}
