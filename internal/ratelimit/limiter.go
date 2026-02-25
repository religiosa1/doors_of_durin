package ratelimit

import (
	"net"
	"sync"
	"time"
)

type Config struct {
	MaxAttempts int
	Window      time.Duration
	FailDelay   time.Duration
}

type entry struct {
	count   int
	resetAt time.Time
}

type Limiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	cfg     Config
	now     func() time.Time
}

func New(cfg Config) *Limiter {
	return &Limiter{
		entries: make(map[string]*entry),
		cfg:     cfg,
		now:     time.Now,
	}
}

func (l *Limiter) IsBlocked(rawIP string) bool {
	ip := normalizeIP(rawIP)
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.getOrCreate(ip)
	return e.count >= l.cfg.MaxAttempts
}

func (l *Limiter) RecordFailure(rawIP string) {
	ip := normalizeIP(rawIP)
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.getOrCreate(ip)
	e.count++
}

func (l *Limiter) FailDelay() time.Duration {
	return l.cfg.FailDelay
}

// getOrCreate returns the entry for ip, lazily resetting an expired window.
// Must be called with l.mu held.
func (l *Limiter) getOrCreate(ip string) *entry {
	now := l.now()
	e, ok := l.entries[ip]
	if !ok || now.After(e.resetAt) {
		e = &entry{resetAt: now.Add(l.cfg.Window)}
		l.entries[ip] = e
	}
	return e
}

func normalizeIP(rawIP string) string {
	host, _, err := net.SplitHostPort(rawIP)
	if err != nil {
		return rawIP
	}
	return host
}
