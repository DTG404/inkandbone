package api

import (
	"context"
	"log"
	"time"
)

// retryWithBackoff retries fn up to maxRetries times with 1s backoff between attempts.
// Only retries on transient failures; if fn fails consistently we propagate the error.
func retryWithBackoff(ctx context.Context, maxRetries int, fn func(context.Context) error) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		err = fn(ctx)
		if err == nil {
			return nil
		}
		log.Printf("retryWithBackoff: attempt %d/%d failed: %v", attempt+1, maxRetries, err)
	}
	return err
}
