package timer

import (
	"sync"
	"time"

	"dfnotes-go/internal/config"
)

type Service interface {
	Start()
	Stop()
	ResetFull()
	ResetPartial()
	Snooze(minutes int)
	Pause()
	Resume()
	IsPaused() bool
}

type resetKind int

const (
	resetFull    resetKind = iota
	resetPartial
	resetSnooze
)

type resetMsg struct {
	kind    resetKind
	minutes int // used for snooze
}

type service struct {
	cfg    *config.Config
	emitFn func(minutesElapsed int)

	mu              sync.Mutex
	paused          bool
	running         bool
	intervalMinutes int // snapshot taken at Start(); avoids cfg reads inside goroutine

	resetCh chan resetMsg
	stopCh  chan struct{}
}

// New returns a Service backed by cfg. emitFn is called each time the reminder fires.
func New(cfg *config.Config, emitFn func(minutesElapsed int)) Service {
	return &service{
		cfg:    cfg,
		emitFn: emitFn,
	}
}

func (s *service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.DocReminderEnabled {
		return
	}
	if s.running {
		return
	}

	s.paused = false
	s.running = true
	s.intervalMinutes = s.cfg.DocReminderIntervalMinutes
	s.resetCh = make(chan resetMsg, 1)
	s.stopCh = make(chan struct{})

	go s.loop(s.stopCh, s.resetCh, s.intervalMinutes)
}

func (s *service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	s.paused = false
	close(s.stopCh)
}

func (s *service) ResetFull() {
	s.sendReset(resetMsg{kind: resetFull})
}

func (s *service) ResetPartial() {
	s.sendReset(resetMsg{kind: resetPartial})
}

func (s *service) Snooze(minutes int) {
	if minutes < 1 {
		minutes = 1
	}
	s.sendReset(resetMsg{kind: resetSnooze, minutes: minutes})
}

func (s *service) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = true
}

func (s *service) Resume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = false
}

func (s *service) IsPaused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.paused
}

// sendReset sends a reset message non-blocking; a no-op if the goroutine is not running.
// If a reset is already pending in the channel buffer, the new one is silently dropped --
// rapid successive resets (e.g. during snooze) are intentionally coalesced.
func (s *service) sendReset(msg resetMsg) {
	s.mu.Lock()
	running := s.running
	ch := s.resetCh
	s.mu.Unlock()

	if !running {
		return
	}
	select {
	case ch <- msg:
	default:
	}
}

func (s *service) loop(stopCh <-chan struct{}, resetCh <-chan resetMsg, intervalMinutes int) {
	full := func() time.Duration { return time.Duration(intervalMinutes) * time.Minute }
	half := func() time.Duration {
		h := intervalMinutes / 2
		if h < 1 {
			h = 1
		}
		return time.Duration(h) * time.Minute
	}

	ticker := time.NewTicker(full())
	lastReset := time.Now()

	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return

		case msg := <-resetCh:
			ticker.Stop()

			if msg.kind == resetSnooze {
				// During snooze the goroutine is blocked; reset signals sent while snoozed
				// are silently dropped (channel buffer size 1, sendReset is non-blocking).
				snoozeTimer := time.NewTimer(time.Duration(msg.minutes) * time.Minute)
				select {
				case <-stopCh:
					snoozeTimer.Stop()
					return
				case <-snoozeTimer.C:
				}
				lastReset = time.Now()
				ticker = time.NewTicker(full())
			} else {
				lastReset = time.Now()
				if msg.kind == resetPartial {
					ticker = time.NewTicker(half())
				} else {
					ticker = time.NewTicker(full())
				}
			}

		case <-ticker.C:
			s.mu.Lock()
			paused := s.paused
			s.mu.Unlock()

			if !paused {
				elapsed := int(time.Since(lastReset).Minutes())
				s.emitFn(elapsed)
			}
		}
	}
}
