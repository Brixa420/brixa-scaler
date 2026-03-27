package main

import (
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:      5,
		InitialDelayMs:   100,
		MaxDelayMs:       1000,
		BackoffMultiplier: 2.0,
		Jitter:           false,
	}

	delays := []time.Duration{}
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		delays = append(delays, cfg.ExponentialBackoff(attempt))
	}

	// Check exponential growth
	if delays[0] != 100*time.Millisecond {
		t.Errorf("expected 100ms, got %v", delays[0])
	}
	if delays[1] != 200*time.Millisecond {
		t.Errorf("expected 200ms, got %v", delays[1])
	}
	if delays[2] != 400*time.Millisecond {
		t.Errorf("expected 400ms, got %v", delays[2])
	}
	if delays[3] != 800*time.Millisecond {
		t.Errorf("expected 800ms, got %v", delays[3])
	}
	// Should cap at MaxDelayMs
	if delays[4] != 1000*time.Millisecond {
		t.Errorf("expected 1000ms (capped), got %v", delays[4])
	}
}

func TestRetrySucceeds(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:      3,
		InitialDelayMs:   10,
		MaxDelayMs:       100,
		BackoffMultiplier: 2.0,
		Jitter:           false,
	}

	callCount := 0
	result, err := Retry(cfg, func() (int, error) {
		callCount++
		if callCount < 3 {
			return 0, &RetryableError{Err: fmt.Errorf("temporary error")}
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetryExhausted(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:      3,
		InitialDelayMs:   10,
		MaxDelayMs:       100,
		BackoffMultiplier: 2.0,
		Jitter:           false,
	}

	_, err := Retry(cfg, func() (int, error) {
		return 0, &RetryableError{Err: fmt.Errorf("persistent error")}
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRetryNonRetryable(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:      3,
		InitialDelayMs:   10,
		MaxDelayMs:       100,
		BackoffMultiplier: 2.0,
		Jitter:           false,
	}

	callCount := 0
	_, err := Retry(cfg, func() (int, error) {
		callCount++
		return 0, fmt.Errorf("non-retryable error")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry for non-retryable), got %d", callCount)
	}
}

func TestCircuitBreakerClosed(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, time.Second)
	
	err := cb.Execute(func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cb.GetState() != "closed" {
		t.Errorf("expected closed state, got %s", cb.GetState())
	}
}

func TestCircuitBreakerOpens(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, time.Second)
	
	// Fail twice to open
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}
	
	if cb.GetState() != "open" {
		t.Errorf("expected open state, got %s", cb.GetState())
	}
	
	// Should fail when circuit is open
	err := cb.Execute(func() error {
		return nil
	})
	
	if err == nil {
		t.Error("expected error when circuit is open")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 50*time.Millisecond)
	
	// Fail to open
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("error")
		})
	}
	
	// Wait for timeout
	time.Sleep(100 * time.Millisecond)
	
	// Should be half-open now
	if cb.GetState() != "half-open" {
		t.Errorf("expected half-open state, got %s", cb.GetState())
	}
}

func TestMetricsTPS(t *testing.T) {
	m := NewMetricsCollector()
	
	// Simulate 100 txs batched over 10 seconds
	m.StartTime = time.Now().Add(-10 * time.Second)
	m.IncTxBatched(100)
	
	tps := m.GetTPS()
	if tps < 9.9 || tps > 10.1 {
		t.Errorf("expected ~10 TPS, got %.2f", tps)
	}
}

func TestMetricsPercentile(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	
	p50 := GetPercentile(values, 50)
	if p50 != 5 {
		t.Errorf("expected p50=5, got %.0f", p50)
	}
	
	p99 := GetPercentile(values, 99)
	if p99 != 10 {
		t.Errorf("expected p99=10, got %.0f", p99)
	}
}