package errors

import (
	"math/rand"
	"time"
)

// RetryController implements exponential backoff with jitter for retry logic.
type RetryController struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	maxRetries   int
}

// NewRetryController creates a new retry controller with default settings.
// Default: initial delay 10ms, max delay 1s, max retries 5
func NewRetryController() *RetryController {
	return &RetryController{
		initialDelay: 10 * time.Millisecond,
		maxDelay:     1 * time.Second,
		maxRetries:   5,
	}
}

// Retry executes a function with retry logic based on error classification.
func (rc *RetryController) Retry(fn func() error, classifier *Classifier) error {
	var lastErr error

	for attempt := 0; attempt <= rc.maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		category := classifier.Classify(err)

		// Don't retry permanent or validation errors
		if !classifier.ShouldRetry(category) {
			return err
		}

		// Don't retry on last attempt
		if attempt >= rc.maxRetries {
			return err
		}

		// Calculate delay with exponential backoff and jitter
		delay := rc.calculateDelay(attempt)
		time.Sleep(delay)
	}

	return lastErr
}

// calculateDelay calculates the delay for a given attempt using exponential backoff + jitter.
func (rc *RetryController) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: delay = initialDelay * 2^attempt
	delay := rc.initialDelay * time.Duration(1<<uint(attempt))

	// Cap at max delay
	if delay > rc.maxDelay {
		delay = rc.maxDelay
	}

	// Add jitter: ±25% random variation
	jitter := time.Duration(float64(delay) * 0.25 * (rand.Float64()*2 - 1))
	delay += jitter

	// Ensure non-negative
	if delay < 0 {
		delay = rc.initialDelay
	}

	return delay
}
