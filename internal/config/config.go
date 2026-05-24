package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DatabasePath              string `json:"database_path"`
	Theme                     string `json:"theme"`
	BackupEnabled             bool   `json:"backup_enabled"`
	BackupDestPath            string `json:"backup_dest_path"`
	BackupIntervalH           int    `json:"backup_interval_hours"`
	BackupKeepCount           int    `json:"backup_keep_count"`
	LastBackupAt              string `json:"last_backup_at"`     // ISO 8601 UTC; empty if never
	LastBackupStatus          string `json:"last_backup_status"` // "success", "failed", or ""
	DocReminderEnabled         bool   `json:"doc_reminder_enabled"`
	DocReminderIntervalMinutes int    `json:"doc_reminder_interval_minutes"`
}

func DefaultConfig() Config {
	return Config{
		Theme:                      "forensic-dark",
		BackupEnabled:              false,
		BackupIntervalH:            6,
		BackupKeepCount:            10,
		DocReminderEnabled:          false,
		DocReminderIntervalMinutes: 30,
	}
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(base, "dfnotes-go"), nil
}

func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := DefaultConfig()
		return &cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// TaskTemplate represents a named set of tasks that can be applied to a case.
type TaskTemplate struct {
	Name  string         `json:"name"`
	Tasks []TemplateTask `json:"tasks"`
}

// TemplateTask is a single task within a template.
type TemplateTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// TemplatesFile is the root structure of templates.json.
type TemplatesFile struct {
	Templates []TaskTemplate `json:"templates"`
}

func templatesPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "templates.json"), nil
}

// LoadTemplates reads templates from ~/.config/dfnotes-go/templates.json.
// Returns an empty TemplatesFile if the file does not exist.
func LoadTemplates() (*TemplatesFile, error) {
	path, err := templatesPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &TemplatesFile{Templates: []TaskTemplate{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read templates: %w", err)
	}
	var tf TemplatesFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	if tf.Templates == nil {
		tf.Templates = []TaskTemplate{}
	}
	return &tf, nil
}

// SaveTemplates writes templates to ~/.config/dfnotes-go/templates.json atomically.
func SaveTemplates(tf *TemplatesFile) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	path, err := templatesPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal templates: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write templates: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("save templates: %w", err)
	}
	return nil
}

func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
