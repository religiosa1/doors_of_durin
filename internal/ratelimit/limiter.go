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
	stop    chan struct{}
}

func New(cfg Config) *Limiter {
	l := &Limiter{
		entries: make(map[string]*entry),
		cfg:     cfg,
		now:     time.Now,
		stop:    make(chan struct{}),
	}
	if cfg.Window > 0 {
		go l.cleanup()
	}
	return l
}

func (l *Limiter) Stop() {
	if l.stop != nil {
		close(l.stop)
	}
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(l.cfg.Window)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.purgeExpired()
		case <-l.stop:
			return
		}
	}
}

func (l *Limiter) purgeExpired() {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	for ip, e := range l.entries {
		if now.After(e.resetAt) {
			delete(l.entries, ip)
		}
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
