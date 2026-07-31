package checker

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"orbital-sentinel/internal/config"
	"orbital-sentinel/internal/logger"
)

type AlertDispatcher interface {
	SendCritical(name, url string, failures int, reason string)
	SendRecovery(name, url string, latency time.Duration)
}

type SweepRecorder interface {
	Record(sweepID int, timestamp time.Time, target, status string, statusCode int, latencyMs int64, errMsg string) error
}

type TargetState struct {
	Config       config.Target
	FailureCount int
	IsDown       bool
	LastLatency  time.Duration
	mu           sync.Mutex
}

type SweepEngine struct {
	States    []*TargetState
	Alerter   AlertDispatcher
	Recorder  SweepRecorder
	Threshold int
}

func NewSweepEngine(targets []config.Target, a AlertDispatcher, r SweepRecorder, threshold int) *SweepEngine {
	states := make([]*TargetState, len(targets))
	for i, t := range targets {
		states[i] = &TargetState{Config: t}
	}
	return &SweepEngine{
		States:    states,
		Alerter:   a,
		Recorder:  r,
		Threshold: threshold,
	}
}

func (e *SweepEngine) RunSweep(iteration int) {
	logger.ScanSweepStart(iteration, len(e.States))
	var wg sync.WaitGroup
	for _, state := range e.States {
		wg.Add(1)
		go func(s *TargetState) {
			defer wg.Done()
			e.checkTarget(s, iteration)
		}(state)
	}
	wg.Wait()

	up, down, unstable := e.statusSummary()
	logger.SweepSummary(up, down, unstable)
}

func (e *SweepEngine) checkTarget(s *TargetState, sweepID int) {
	s.mu.Lock()
	cfg := s.Config
	s.mu.Unlock()

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	client := &http.Client{Timeout: timeout}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, nil)
	if err != nil {
		e.handleFailure(s, sweepID, fmt.Sprintf("request error: %v", err))
		return
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
	e.handleFailure(s, sweepID, fmt.Sprintf("no response: %v", err))
		return
	}
	defer resp.Body.Close()

	if !cfg.IsStatusAllowed(resp.StatusCode) {
	e.handleFailure(s, sweepID, fmt.Sprintf("status %d (expected %d)", resp.StatusCode, cfg.ExpectedStatus))
		return
	}

	e.handleSuccess(s, sweepID, latency, resp.StatusCode)
}

func (e *SweepEngine) handleSuccess(s *TargetState, sweepID int, latency time.Duration, statusCode int) {
	s.mu.Lock()
	s.LastLatency = latency
	wasDown := s.IsDown
	s.IsDown = false
	s.FailureCount = 0
	s.mu.Unlock()

	name := s.Config.Name
	if e.Recorder != nil {
		_ = e.Recorder.Record(sweepID, time.Now(), name, "UP", statusCode, latency.Milliseconds(), "")
	}

	if wasDown {
		logger.NodeRecovered(name, latency)
		e.Alerter.SendRecovery(name, s.Config.URL, latency)
	} else {
		logger.NodeStable(name, latency, statusCode)
	}
}

func (e *SweepEngine) handleFailure(s *TargetState, sweepID int, reason string) {
	s.mu.Lock()
	s.FailureCount++
	count := s.FailureCount
	wasDown := s.IsDown
	if count >= e.Threshold {
		s.IsDown = true
	}
	s.mu.Unlock()

	name := s.Config.Name
	if e.Recorder != nil {
		_ = e.Recorder.Record(sweepID, time.Now(), name, "DOWN", 0, 0, reason)
	}

	if count < e.Threshold {
		logger.NodeWarning(name, count, e.Threshold, reason)
	} else if !wasDown {
		logger.NodeCritical(name, reason)
		e.Alerter.SendCritical(name, s.Config.URL, count, reason)
	} else {
		logger.NodeCritical(name, reason)
	}
}

func (e *SweepEngine) StatusSnapshot() []map[string]interface{} {
	var snap []map[string]interface{}
	for _, s := range e.States {
		s.mu.Lock()
		status := "UP"
		if s.IsDown {
			status = "DOWN"
		} else if s.FailureCount > 0 {
			status = "UNSTABLE"
		}
		snap = append(snap, map[string]interface{}{
			"name":          s.Config.Name,
			"url":           s.Config.URL,
			"status":        status,
			"status_code":   s.Config.ExpectedStatus,
			"latency_ms":    s.LastLatency.Milliseconds(),
			"failure_count": s.FailureCount,
			"threshold":     e.Threshold,
			"is_down":       s.IsDown,
		})
		s.mu.Unlock()
	}
	return snap
}

func (e *SweepEngine) statusSummary() (up int, down int, unstable int) {
	for _, s := range e.States {
		s.mu.Lock()
		if s.IsDown {
			down++
		} else if s.FailureCount > 0 {
			unstable++
		} else {
			up++
		}
		s.mu.Unlock()
	}
	return
}
