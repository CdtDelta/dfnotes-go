package database

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyAndVerify copies src to dst using a temp file + atomic rename, then
// verifies that the SHA-256 of both files match. If dst already exists it
// is overwritten only after a successful copy and verify.
func CopyAndVerify(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	srcHash, err := sha256File(src)
	if err != nil {
		return fmt.Errorf("hash source file: %w", err)
	}

	tmp := dst + ".tmp"
	if err := copyFile(src, tmp); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("copy file: %w", err)
	}

	dstHash, err := sha256File(tmp)
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("hash destination file: %w", err)
	}

	if srcHash != dstHash {
		os.Remove(tmp)
		return fmt.Errorf("copy integrity check failed: SHA-256 mismatch")
	}

	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename to destination: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ValidateSchema opens the file at path and checks for the presence of core
// dfnotes-go tables. Returns an error if the file is not a valid dfnotes-go database.
func ValidateSchema(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	required := []string{"users", "cases", "note_blocks", "evidence_items", "ioc_entries"}
	for _, table := range required {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			return fmt.Errorf("not a dfnotes-go database: missing table %q", table)
		}
	}
	return nil
}
