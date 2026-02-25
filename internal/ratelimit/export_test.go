package ratelimit

import "time"

func NewWithClock(cfg Config, now func() time.Time) *Limiter {
	return &Limiter{
		entries: make(map[string]*entry),
		cfg:     cfg,
		now:     now,
	}
}
