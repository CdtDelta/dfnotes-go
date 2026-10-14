package database

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSchemaCurrentVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "current.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	db.Close()

	if err := ValidateSchema(path); err != nil {
		t.Errorf("ValidateSchema on migrated database: %v", err)
	}
}

// ValidateSchema runs before migrations, so a database that still has
// ioc_entries must pass too.
func TestValidateSchemaPreIndicatorVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v15.db")
	db := openAtVersion(t, path, preIndicatorVersion)
	if v := schemaVersion(t, db.DB); v != preIndicatorVersion {
		db.Close()
		t.Fatalf("schema version = %d, want %d", v, preIndicatorVersion)
	}
	db.Close()

	if err := ValidateSchema(path); err != nil {
		t.Errorf("ValidateSchema on v%d database: %v", preIndicatorVersion, err)
	}
}

func TestValidateSchemaRejectsForeignDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "foreign.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := raw.Exec("CREATE TABLE something_else (id INTEGER PRIMARY KEY)"); err != nil {
		raw.Close()
		t.Fatalf("create table: %v", err)
	}
	raw.Close()

	err = ValidateSchema(path)
	if err == nil {
		t.Fatal("ValidateSchema accepted a database with none of the required tables")
	}
	if !strings.Contains(err.Error(), "not a dfnotes-go database") {
		t.Errorf("error = %q, want it to contain %q", err, "not a dfnotes-go database")
	}
}
