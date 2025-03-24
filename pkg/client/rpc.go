package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// callWithRetry executes an RPC call with retry logic
func (c *Client) callWithRetry(ctx context.Context, operation string, fn func(context.Context) error) error {
	var lastErr error
	
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Create a context with timeout
		callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
		
		// Call the function
		err := fn(callCtx)
		
		// Cancel the context
		cancel()
		
		// If successful or not retryable, return the result
		if err == nil || !isRetryableError(err) {
			return err
		}
		
		// Store the last error
		lastErr = err
		
		// If this was the last attempt, break
		if attempt == c.config.MaxRetries {
			break
		}
		
		// Calculate retry delay with exponential backoff
		delay := c.config.RetryDelay * time.Duration(float64(attempt+1)*c.config.BackoffFactor)
		
		// Wait for the retry delay or until the context is canceled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to the next attempt
		}
	}
	
	// Return the last error
	return fmt.Errorf("operation %s failed after %d attempts: %w", operation, c.config.MaxRetries+1, lastErr)
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	// If it's a context error, it's not retryable
	if err == context.DeadlineExceeded || err == context.Canceled {
		return false
	}
	
	// Check gRPC error codes
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.Unavailable, codes.ResourceExhausted, codes.Aborted:
			// Server is unavailable, resource exhausted, or transaction aborted
			return true
		case codes.Internal, codes.Unknown:
			// Internal server error or unknown error
			return true
		default:
			// Other errors are not retryable
			return false
		}
	}
	
	// Default to not retryable
	return false
}
