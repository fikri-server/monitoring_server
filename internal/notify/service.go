package notify

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"monitoring_server/internal/battery"
)

const retryCooldown = time.Minute

type Status struct {
	Configured bool    `json:"configured"`
	Threshold  float64 `json:"threshold"`
	LastSent   string  `json:"last_sent,omitempty"`
	LastError  string  `json:"last_error,omitempty"`
	Armed      bool    `json:"armed"`
}

type Service struct {
	mu       sync.Mutex
	cfg      Config
	send     Sender
	hostname string

	notified    bool
	lastSent    time.Time
	lastAttempt time.Time
	lastErr     string
}

func NewService(cfg Config, send Sender) *Service {
	host, _ := os.Hostname()
	return &Service{cfg: cfg, send: send, hostname: host}
}

// Check sends at most one email per discharge episode. A new episode starts
// when the battery stops discharging or climbs back above the threshold.
func (s *Service) Check(info battery.BatteryInfo, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Configured() {
		return
	}

	if info.Status != "Discharging" || info.Percentage >= s.cfg.Threshold {
		s.notified = false
		return
	}
	if s.notified || info.Percentage <= 0 {
		return
	}
	// The cooldown only throttles retries after a failed send; a fresh
	// episode may send immediately.
	if s.lastErr != "" && !s.lastAttempt.IsZero() && now.Sub(s.lastAttempt) < retryCooldown {
		return
	}

	s.lastAttempt = now
	msg := buildMessage(s.cfg.User, s.cfg.Recipient, s.hostname, info, now)
	if err := s.send(s.cfg.User, s.cfg.Password, s.cfg.Recipient, msg); err != nil {
		s.lastErr = sanitize(err.Error(), s.cfg.User, s.cfg.Password, s.cfg.Recipient)
		return
	}

	s.notified = true
	s.lastSent = now
	s.lastErr = ""
}

func (s *Service) SendTest() error {
	s.mu.Lock()
	if !s.cfg.Configured() {
		s.mu.Unlock()
		return errors.New("email notification not configured")
	}
	cfg, send, hostname := s.cfg, s.send, s.hostname
	s.mu.Unlock()

	msg := buildTestMessage(cfg.User, cfg.Recipient, hostname, time.Now())
	if err := send(cfg.User, cfg.Password, cfg.Recipient, msg); err != nil {
		return errors.New(sanitize(err.Error(), cfg.User, cfg.Password, cfg.Recipient))
	}
	return nil
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := Status{
		Configured: s.cfg.Configured(),
		Threshold:  s.cfg.Threshold,
		LastError:  s.lastErr,
		Armed:      s.cfg.Configured() && !s.notified,
	}
	if !s.lastSent.IsZero() {
		st.LastSent = s.lastSent.Format(time.RFC3339)
	}
	return st
}

func sanitize(msg string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			msg = strings.ReplaceAll(msg, secret, "[redacted]")
		}
	}
	return msg
}
