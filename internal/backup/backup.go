package backup

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PerformBackup copies the database file at dbPath into destDir.
// The backup file is named dfnotes-go_backup_<ISO8601UTC>.db.
// An atomic write is used: data is written to a temp file, verified, then renamed.
// Returns the path of the created backup file.
func PerformBackup(dbPath string, destDir string) (string, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("source database not found: %s", dbPath)
		}
		return "", fmt.Errorf("stat source database: %w", err)
	}

	if _, err := os.Stat(destDir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("backup destination directory does not exist: %s", destDir)
		}
		return "", fmt.Errorf("stat destination directory: %w", err)
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	filename := "dfnotes-go_backup_" + ts + ".db"
	destPath := filepath.Join(destDir, filename)
	tmp := destPath + ".tmp"

	srcHash, err := sha256File(dbPath)
	if err != nil {
		return "", fmt.Errorf("hash source: %w", err)
	}

	if err := copyFile(dbPath, tmp); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("copy database: %w", err)
	}

	dstHash, err := sha256File(tmp)
	if err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("hash destination: %w", err)
	}

	if srcHash != dstHash {
		os.Remove(tmp)
		return "", fmt.Errorf("backup integrity check failed: SHA-256 mismatch")
	}

	if err := os.Rename(tmp, destPath); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("rename backup file: %w", err)
	}

	return destPath, nil
}

// PruneBackups removes the oldest dfnotes-go_backup_*.db files in destDir
// until only keepCount files remain.
func PruneBackups(destDir string, keepCount int) error {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return fmt.Errorf("read backup directory: %w", err)
	}

	var backups []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "dfnotes-go_backup_") && strings.HasSuffix(e.Name(), ".db") {
			backups = append(backups, filepath.Join(destDir, e.Name()))
		}
	}

	sort.Strings(backups) // ISO 8601 names sort lexicographically = chronologically

	if len(backups) <= keepCount {
		return nil
	}

	toDelete := backups[:len(backups)-keepCount]
	for _, path := range toDelete {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove old backup %s: %w", path, err)
		}
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
