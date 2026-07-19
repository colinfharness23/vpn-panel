package controller

import (
	"strings"
	"sync"
	"time"
)

const (
	loginLimitMaxFailures = 5
	loginLimitWindow      = 5 * time.Minute
	loginLimitCooldown    = 15 * time.Minute
)

var defaultLoginLimiter = newLoginLimiter(loginLimitMaxFailures, loginLimitWindow, loginLimitCooldown)

type loginLimiter struct {
	mu          sync.Mutex
	now         func() time.Time
	enabled     bool
	maxFailures int
	window      time.Duration
	cooldown    time.Duration
	attempts    map[string]*loginLimitRecord
}

type loginLimitRecord struct {
	failures     []time.Time
	blockedUntil time.Time
}

func newLoginLimiter(maxFailures int, window, cooldown time.Duration) *loginLimiter {
	return &loginLimiter{
		now:         time.Now,
		enabled:     true,
		maxFailures: maxFailures,
		window:      window,
		cooldown:    cooldown,
		attempts:    make(map[string]*loginLimitRecord),
	}
}

func (l *loginLimiter) configure(enabled bool, maxFailures int, window, cooldown time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if maxFailures < 1 {
		maxFailures = loginLimitMaxFailures
	}
	if window <= 0 {
		window = loginLimitWindow
	}
	if cooldown <= 0 {
		cooldown = loginLimitCooldown
	}
	l.enabled = enabled
	l.maxFailures = maxFailures
	l.window = window
	l.cooldown = cooldown
	if !enabled {
		l.attempts = make(map[string]*loginLimitRecord)
	}
}

func (l *loginLimiter) allow(ip, username string) (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return time.Time{}, true
	}

	key := loginLimitKey(ip, username)
	record := l.attempts[key]
	if record == nil {
		return time.Time{}, true
	}
	now := l.now()
	if now.Before(record.blockedUntil) {
		return record.blockedUntil, false
	}
	record.blockedUntil = time.Time{}
	record.failures = pruneLoginFailures(record.failures, now.Add(-l.window))
	if len(record.failures) == 0 {
		delete(l.attempts, key)
	}
	return time.Time{}, true
}

func (l *loginLimiter) registerFailure(ip, username string) (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return time.Time{}, false
	}

	key := loginLimitKey(ip, username)
	record := l.attempts[key]
	if record == nil {
		record = &loginLimitRecord{}
		l.attempts[key] = record
	}
	now := l.now()
	record.failures = pruneLoginFailures(record.failures, now.Add(-l.window))
	record.failures = append(record.failures, now)
	if len(record.failures) >= l.maxFailures {
		record.failures = nil
		record.blockedUntil = now.Add(l.cooldown)
		return record.blockedUntil, true
	}
	return time.Time{}, false
}

func (l *loginLimiter) registerSuccess(ip, username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return
	}
	delete(l.attempts, loginLimitKey(ip, username))
}

func loginLimitKey(ip, username string) string {
	return strings.TrimSpace(ip) + "\x00" + strings.ToLower(strings.TrimSpace(username))
}

func pruneLoginFailures(failures []time.Time, cutoff time.Time) []time.Time {
	keepFrom := 0
	for keepFrom < len(failures) && failures[keepFrom].Before(cutoff) {
		keepFrom++
	}
	return failures[keepFrom:]
}
