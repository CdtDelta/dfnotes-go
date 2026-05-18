package backup

import (
	"fmt"
	"sync"
	"time"

	"dfnotes-go/internal/config"
)

type Status struct {
	LastBackupTime   string `json:"last_backup_time"`
	LastBackupStatus string `json:"last_backup_status"`
	IsRunning        bool   `json:"is_running"`
}

type Scheduler struct {
	mu        sync.Mutex
	cfg       *config.Config
	dbPath    string
	onFailure func(err error)
	ticker    *time.Ticker
	stop      chan struct{}
	lastTime  string
	lastOK    bool
	hasRun    bool
	running   bool
}

func NewScheduler(cfg *config.Config, dbPath string, onFailure func(err error)) *Scheduler {
	return &Scheduler{
		cfg:       cfg,
		dbPath:    dbPath,
		onFailure: onFailure,
		stop:      make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	go s.run()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stop)
	s.stop = make(chan struct{})
}

func (s *Scheduler) RunNow() error {
	s.mu.Lock()
	cfg := s.cfg
	dbPath := s.dbPath
	s.mu.Unlock()

	if cfg.BackupDestPath == "" {
		return fmt.Errorf("no backup destination configured")
	}
	_, err := PerformBackup(dbPath, cfg.BackupDestPath)

	now := time.Now().UTC().Format(time.RFC3339)
	cfg.LastBackupAt = now
	if err == nil {
		cfg.LastBackupStatus = "success"
	} else {
		cfg.LastBackupStatus = "failed"
	}
	config.Save(cfg) //nolint:errcheck

	s.mu.Lock()
	s.hasRun = true
	s.lastTime = now
	s.lastOK = err == nil
	s.mu.Unlock()

	if err == nil {
		PruneBackups(cfg.BackupDestPath, cfg.BackupKeepCount) //nolint:errcheck
	}
	return err
}

func (s *Scheduler) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{IsRunning: s.running}
	if s.hasRun {
		st.LastBackupTime = s.lastTime
		if s.lastOK {
			st.LastBackupStatus = "success"
		} else {
			st.LastBackupStatus = "failed"
		}
	} else {
		st.LastBackupStatus = "never"
	}
	return st
}

func (s *Scheduler) run() {
	s.mu.Lock()
	interval := time.Duration(s.cfg.BackupIntervalH) * time.Hour
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	s.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.RunNow(); err != nil && s.onFailure != nil {
				s.onFailure(err)
			}
		case <-s.stop:
			return
		}
	}
}
