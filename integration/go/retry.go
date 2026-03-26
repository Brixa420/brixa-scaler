package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts      int     // Maximum number of retry attempts
	InitialDelayMs   int     // Initial delay in milliseconds
	MaxDelayMs       int     // Maximum delay in milliseconds
	BackoffMultiplier float64 // Multiplier for exponential backoff
	Jitter           bool    // Whether to add jitter
}

// RetryableError is an error that can be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

// IsRetryable returns true if the error can be retried
func IsRetryable(err error) bool {
	if _, ok := err.(*RetryableError); ok {
		return true
	}
	// Add logic for specific retryable errors (network, timeout, etc.)
	return false
}

// ExponentialBackoff calculates the next delay with optional jitter
func (r *RetryConfig) ExponentialBackoff(attempt int) time.Duration {
	delay := time.Duration(r.InitialDelayMs) * time.Millisecond
	
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * r.BackoffMultiplier)
		if delay > time.Duration(r.MaxDelayMs)*time.Millisecond {
			delay = time.Duration(r.MaxDelayMs) * time.Millisecond
		}
	}
	
	if r.Jitter {
		// Add +/- 25% jitter
		jitterFactor := 0.75 + rand.Float64()*0.5
		delay = time.Duration(float64(delay) * jitterFactor)
	}
	
	return delay
}

// Retry executes fn with exponential backoff retry logic
func Retry[T any](cfg *RetryConfig, fn func() (T, error)) (T, error) {
	var lastErr error
	
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Don't retry if not a retryable error
		if !IsRetryable(err) {
			return result, err
		}
		
		// Check if this was the last attempt
		if attempt == cfg.MaxAttempts - 1 {
			break
		}
		
		// Calculate delay and wait
		delay := cfg.ExponentialBackoff(attempt)
		time.Sleep(delay)
	}
	
	var zero T
	return zero, fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
}

// RetryContext executes fn with context and retry logic
func RetryContext[T any](ctx context.Context, cfg *RetryConfig, fn func(context.Context) (T, error)) (T, error) {
	var lastErr error
	
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Check for context cancellation
		if ctx.Err() != nil {
			var zero T
			return zero, fmt.Errorf("context cancelled: %w", ctx.Err())
		}
		
		// Don't retry if not a retryable error
		if !IsRetryable(err) {
			return result, err
		}
		
		// Check if this was the last attempt
		if attempt == cfg.MaxAttempts - 1 {
			break
		}
		
		// Calculate delay and wait
		delay := cfg.ExponentialBackoff(attempt)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		}
	}
	
	var zero T
	return zero, fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
}

// RetryWithCondition retries only when the condition returns true
func RetryWithCondition[T any](cfg *RetryConfig, fn func() (T, error), shouldRetry func(error) bool) (T, error) {
	var lastErr error
	
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Check custom retry condition
		if !shouldRetry(err) {
			return result, err
		}
		
		// Check if this was the last attempt
		if attempt == cfg.MaxAttempts - 1 {
			break
		}
		
		// Calculate delay and wait
		delay := cfg.ExponentialBackoff(attempt)
		time.Sleep(delay)
	}
	
	var zero T
	return zero, fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
}

// BackoffWithMaxRetries is a convenient function for common retry scenarios
func BackoffWithMaxRetries(fn func() error, maxRetries int) error {
	cfg := &RetryConfig{
		MaxAttempts:      maxRetries,
		InitialDelayMs:   1000,
		MaxDelayMs:       60000,
		BackoffMultiplier: 2.0,
		Jitter:           true,
	}
	
	_, err := Retry(cfg, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	
	return err
}

// CircuitBreaker prevents cascading failures
type CircuitBreaker struct {
	failures         int
	successes        int
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	lastFailure      time.Time
	state            string // closed, open, half-open
	mu               sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		state:            "closed",
	}
}

// Execute runs fn through the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	// Check if circuit is open
	if cb.state == "open" {
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = "half-open"
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}
	
	// Execute the function
	err := fn()
	
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		
		if cb.failures >= cb.failureThreshold {
			cb.state = "open"
		}
		return err
	}
	
	// Success
	cb.successes++
	cb.failures = 0
	
	if cb.state == "half-open" && cb.successes >= cb.successThreshold {
		cb.state = "closed"
		cb.successes = 0
	}
	
	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// IsHealthy returns true if the circuit breaker is closed
func (cb *CircuitBreaker) IsHealthy() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state == "closed"
}

import "sync" // Add this to the imports in the actual file