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

const (
	migrationDirectory     = "migrations"
	migrationTableName     = "schema_migrations"
	migrationVersionLayout = "20060102150405"

	migrationDriverMySQL    = "mysql"
	migrationDriverPostgres = "pgx"
	migrationDriverSQLite   = "sqlite3"
)

//go:embed migrations
var migrationFiles embed.FS

// Migrate applies pending embedded schema changes for an enabled database.
func Migrate(ctx context.Context, gdb *gorm.DB, cfg Config) error {
	if cfg.Disabled() {
		return nil
	}
	if gdb == nil {
		return errors.New("migration database is nil")
	}
	return runMigrations(ctx, gdb, cfg, migrationFiles)
}

// runMigrations validates a migration source and advances the database to its latest version.
func runMigrations(ctx context.Context, gdb *gorm.DB, cfg Config, migrationFS fs.FS) (err error) {
	hasMigrations, err := validateMigrationFiles(migrationFS)
	if err != nil {
		return err
	}

	sourceDriver, err := iofs.New(migrationFS, migrationDirectory)
	if err != nil {
		return fmt.Errorf("open migration files: %w", err)
	}

	databaseDriver, driverName, err := newMigrationDatabase(gdb, cfg)
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

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("migration interrupted: %w", err)
	}
	return nil
}

// newMigrationDatabase adapts the active Gorm connection for the configured migration driver.
func newMigrationDatabase(gdb *gorm.DB, cfg Config) (migratedatabase.Driver, string, error) {
	driverName, dsn, err := migrationConnectionConfig(cfg)
	if err != nil {
		return nil, "", err
	}

	if driverName == migrationDriverSQLite {
		sqlDB, err := gdb.DB()
		if err != nil {
			return nil, "", fmt.Errorf("get sqlite migration database: %w", err)
		}
		databaseDriver, err := migratesqlite.WithInstance(sqlDB, &migratesqlite.Config{
			MigrationsTable: migrationTableName,
		})
		if err != nil {
			return nil, "", fmt.Errorf("initialize migration database: %w", err)
		}
		return nonClosingMigrationDriver{Driver: databaseDriver}, driverName, nil
	}

	sqlDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, "", fmt.Errorf("open migration database: %w", err)
	}

	var databaseDriver migratedatabase.Driver
	switch driverName {
	case migrationDriverPostgres:
		databaseDriver, err = migratepgx.WithInstance(sqlDB, &migratepgx.Config{
			MigrationsTable: migrationTableName,
		})
	case migrationDriverMySQL:
		databaseDriver, err = migratemysql.WithInstance(sqlDB, &migratemysql.Config{
			MigrationsTable: migrationTableName,
		})
	default:
		err = fmt.Errorf("unsupported migration driver %q", driverName)
	}
	if err != nil {
		return nil, "", errors.Join(fmt.Errorf("initialize migration database: %w", err), sqlDB.Close())
	}
	return databaseDriver, driverName, nil
}

// nonClosingMigrationDriver lets golang-migrate reuse the application's
// SQLite connection without closing it when the short-lived migrator closes.
type nonClosingMigrationDriver struct {
	migratedatabase.Driver
}

// Close preserves the application-owned database connection when migrations release their driver.
func (nonClosingMigrationDriver) Close() error { return nil }

// migrationConnectionConfig normalizes database aliases and DSNs for the migration library.
func migrationConnectionConfig(cfg Config) (driverName, dsn string, err error) {
	switch cfg.Driver {
	case "postgres", "postgresql", "pg":
		return migrationDriverPostgres, cfg.DSN, nil
	case "mysql":
		mysqlConfig, parseErr := mysqldriver.ParseDSN(cfg.DSN)
		if parseErr != nil {
			return "", "", fmt.Errorf("parse mysql migration DSN: %w", parseErr)
		}
		mysqlConfig.MultiStatements = true
		return migrationDriverMySQL, mysqlConfig.FormatDSN(), nil
	case "sqlite", "sqlite3":
		return migrationDriverSQLite, cfg.DSN, nil
	default:
		return "", "", fmt.Errorf("unsupported db.driver %q (want postgres|mysql|sqlite)", cfg.Driver)
	}
}

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
		version := strings.SplitN(entry.Name(), "_", 2)[0]
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
