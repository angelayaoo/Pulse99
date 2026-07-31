package alerter

import (
	"sync"
	"time"

	"orbital-sentinel/internal/logger"
	"orbital-sentinel/internal/notify"
)

type Alerter struct {
	notifiers   []notify.Notifier
	cooldown    time.Duration
	maxRetries  int
	retryBackoff time.Duration

	mu              sync.Mutex
	lastCritical map[string]time.Time
	lastRecovery map[string]time.Time
}

func New(notifiers []notify.Notifier, cooldownSec int, maxRetries int, retryBackoffMs int) *Alerter {
	cooldown := time.Duration(cooldownSec) * time.Second
	if cooldown <= 0 {
		cooldown = 5 * time.Minute
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}
	backoff := time.Duration(retryBackoffMs) * time.Millisecond
	if backoff <= 0 {
		backoff = 2 * time.Second
	}
	return &Alerter{
		notifiers:       notifiers,
		cooldown:        cooldown,
		maxRetries:      maxRetries,
		retryBackoff:    backoff,
		lastCritical:    make(map[string]time.Time),
		lastRecovery:    make(map[string]time.Time),
	}
}

func (a *Alerter) SendCritical(name, url string, failures int, reason string) {
	if len(a.notifiers) == 0 {
		return
	}

	key := name + ":critical"
	a.mu.Lock()
	last, exists := a.lastCritical[key]
	now := time.Now()
	if exists && time.Since(last) < a.cooldown {
		a.mu.Unlock()
		logger.AlertSuppressed(name, "CRITICAL")
		return
	}
	a.lastCritical[key] = now
	a.mu.Unlock()

	p := notify.Payload{
		Name:      name,
		URL:       url,
		Status:    "DOWN",
		Failures:  failures,
		Reason:    reason,
		Timestamp: now,
	}

	a.dispatch("critical", p)
}

func (a *Alerter) SendRecovery(name, url string, latency time.Duration) {
	if len(a.notifiers) == 0 {
		return
	}

	key := name + ":recovery"
	a.mu.Lock()
	last, exists := a.lastRecovery[key]
	now := time.Now()
	if exists && time.Since(last) < a.cooldown {
		a.mu.Unlock()
		logger.AlertSuppressed(name, "RECOVERY")
		return
	}
	a.lastRecovery[key] = now
	a.mu.Unlock()

	p := notify.Payload{
		Name:      name,
		URL:       url,
		Status:    "UP",
		LatencyMs: latency.Milliseconds(),
		Timestamp: now,
	}

	a.dispatch("recovery", p)
}

func (a *Alerter) dispatch(kind string, p notify.Payload) {
	for _, n := range a.notifiers {
		go func(notifier notify.Notifier) {
			a.sendWithRetry(notifier, kind, p)
		}(n)
	}
}

func (a *Alerter) sendWithRetry(n notify.Notifier, kind string, p notify.Payload) {
	var sendFn func(p notify.Payload) error
	switch kind {
	case "critical":
		sendFn = n.SendCritical
	case "recovery":
		sendFn = n.SendRecovery
	default:
		return
	}

	delay := a.retryBackoff
	for attempt := 0; attempt <= a.maxRetries; attempt++ {
		err := sendFn(p)
		if err == nil {
			return
		}
		if attempt < a.maxRetries {
			logger.NotifyRetry(n.Name(), attempt+1, err)
			time.Sleep(delay)
			delay *= 2
		} else {
			logger.NotifyFailed(n.Name(), p.Name, err)
		}
	}
}
