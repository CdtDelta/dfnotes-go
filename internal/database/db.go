package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// goMigrations maps a migration version to Go code that runs in the same
// transaction as that version's SQL file, after the SQL and before the version
// is recorded. Used when a migration needs logic SQL cannot express, such as
// normalizing values with ioc.MatchKey.
var goMigrations = map[int]func(tx *sql.Tx) error{
	16: migrate016IOCIndicators,
}

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := openConn(path)
	if err != nil {
		return nil, err
	}
	if err := db.migrate(path); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// openConn opens the database and applies connection pragmas without running
// migrations, so tests can stop the schema at a specific version.
func openConn(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqlDB.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := sqlDB.Exec(p); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	return &DB{sqlDB}, nil
}

// migrate applies every pending migration.
func (db *DB) migrate(path string) error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	return db.migrateTo(path, len(entries))
}

// migrateTo applies pending migrations up to and including target. path is
// the database file, used to place the pre-migration snapshot.
func (db *DB) migrateTo(path string, target int) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	)`)
	if err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	var current int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	if target > len(entries) {
		return fmt.Errorf("migration target %d exceeds latest version %d", target, len(entries))
	}

	// A brand-new database (version 0) has nothing worth preserving, and
	// skipping the snapshot when nothing is pending keeps normal opens free.
	if current > 0 && target > current {
		snapshot, err := db.snapshot(path, current, target)
		if err != nil {
			return fmt.Errorf("pre-migration snapshot: %w", err)
		}
		log.Printf("Pre-migration snapshot written to %s", snapshot)
	}

	for i, entry := range entries {
		version := i + 1
		if version <= current {
			continue
		}
		if version > target {
			break
		}

		data, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", version, err)
		}

		if _, err := tx.Exec(string(data)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", version, err)
		}

		if hook, ok := goMigrations[version]; ok {
			if err := hook(tx); err != nil {
				tx.Rollback()
				return fmt.Errorf("apply go migration %d: %w", version, err)
			}
		}

		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}

	return nil
}

// snapshot copies the database with VACUUM INTO before migrations run, so a
// one-way schema change can be undone by hand from a consistent copy. Returns
// the snapshot path.
func (db *DB) snapshot(path string, from, to int) (string, error) {
	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	name := fmt.Sprintf("%s.pre-migration-v%d-to-v%d-%s.db",
		stem, from, to, time.Now().UTC().Format("20060102T150405Z"))
	dest := filepath.Join(filepath.Dir(path), name)

	// Bound as a parameter so spaces or quotes in the path need no escaping.
	if _, err := db.Exec("VACUUM INTO ?", dest); err != nil {
		return "", fmt.Errorf("vacuum into %s: %w", dest, err)
	}
	return dest, nil
}

func DefaultDBPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "dfnotes-go", "dfnotes.db")
}

func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
