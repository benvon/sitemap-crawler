package crawler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

// TestRateLimiterSharedAcrossWorkers verifies that the rate limiter
// is shared across workers and caps the total rate, not per-worker rate
func TestRateLimiterSharedAcrossWorkers(t *testing.T) {
	t.Parallel()

	// Create a rate limiter with a low rate for easy testing
	requestsPerSecond := 5
	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond)

	// Number of workers (more than rate limit to ensure contention)
	numWorkers := 10

	// Track when each request happens
	var requestTimes []time.Time
	var mu sync.Mutex

	// Context with timeout to prevent infinite blocking
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()

	// Start multiple workers that all use the same rate limiter
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker tries to make 3 requests
			for j := 0; j < 3; j++ {
				// Wait for rate limiter (this is the key test)
				if err := limiter.Wait(ctx); err != nil {
					// Context timeout or cancellation
					return
				}

				// Record the time of this request
				mu.Lock()
				requestTimes = append(requestTimes, time.Now())
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Analyze the results
	totalDuration := time.Since(start)

	// We should have some requests (even if not all due to timeout)
	assert.Greater(t, len(requestTimes), 0, "Should have made some requests")

	// Calculate actual rate
	if len(requestTimes) > 1 && totalDuration > 0 {
		actualRate := float64(len(requestTimes)-1) / totalDuration.Seconds()
		expectedRate := float64(requestsPerSecond)

		// The actual rate should be close to but not exceed the expected rate
		// Allow some tolerance for timing precision
		tolerance := 1.5 // Allow 50% tolerance due to test timing variability
		assert.LessOrEqual(t, actualRate, expectedRate*tolerance,
			"Actual rate (%.2f req/s) should not significantly exceed expected rate (%.2f req/s)",
			actualRate, expectedRate)

		t.Logf("Expected rate: %.2f req/s", expectedRate)
		t.Logf("Actual rate: %.2f req/s", actualRate)
		t.Logf("Total requests: %d", len(requestTimes))
		t.Logf("Total duration: %v", totalDuration)
	}
}

// TestRateLimiterBehaviorWithMultipleWorkers demonstrates that workers
// coordinate through the shared rate limiter
func TestRateLimiterBehaviorWithMultipleWorkers(t *testing.T) {
	t.Parallel()

	// Very restrictive rate limiter
	limiter := rate.NewLimiter(rate.Limit(2), 1) // 2 requests per second, burst of 1

	numWorkers := 5
	requestsPerWorker := 2

	var mu sync.Mutex
	var requestTimes []time.Time

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				if err := limiter.Wait(ctx); err != nil {
					return
				}

				mu.Lock()
				requestTimes = append(requestTimes, time.Now())
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Verify that requests were spread out over time due to rate limiting
	if len(requestTimes) >= 2 {
		// Check intervals between consecutive requests
		intervals := make([]time.Duration, 0, len(requestTimes)-1)
		for i := 1; i < len(requestTimes); i++ {
			intervals = append(intervals, requestTimes[i].Sub(requestTimes[i-1]))
		}

		// With a rate of 2 req/s, we expect some intervals to be around 500ms
		// (though the first few might be faster due to burst capacity)
		hasSignificantDelay := false
		for _, interval := range intervals {
			if interval > 400*time.Millisecond {
				hasSignificantDelay = true
				break
			}
		}

		assert.True(t, hasSignificantDelay,
			"Should have some requests delayed by rate limiting. Intervals: %v", intervals)

		t.Logf("Request intervals: %v", intervals)
		t.Logf("Total requests completed: %d", len(requestTimes))
		t.Logf("Total time: %v", time.Since(start))
	}
}

// TestSingleWorkerVsMultipleWorkerRates demonstrates that total rate
// is capped regardless of number of workers
func TestSingleWorkerVsMultipleWorkerRates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting comparison test in short mode")
	}

	t.Parallel()

	requestRate := 10 // 10 requests per second
	testDuration := 2 * time.Second

	// Test with single worker
	singleWorkerRequests := testWorkerRequests(t, 1, requestRate, testDuration)

	// Test with multiple workers
	multipleWorkerRequests := testWorkerRequests(t, 5, requestRate, testDuration)

	// Both should complete roughly the same number of requests
	// because the rate limiter caps the total rate
	tolerance := 3 // Allow some variance due to timing
	difference := abs(singleWorkerRequests - multipleWorkerRequests)

	assert.LessOrEqual(t, difference, tolerance,
		"Single worker (%d req) and multiple workers (%d req) should complete similar numbers of requests",
		singleWorkerRequests, multipleWorkerRequests)

	t.Logf("Single worker completed: %d requests", singleWorkerRequests)
	t.Logf("Multiple workers completed: %d requests", multipleWorkerRequests)
}

// Helper function to test request completion with given number of workers
func testWorkerRequests(t *testing.T, numWorkers, requestRate int, duration time.Duration) int {
	limiter := rate.NewLimiter(rate.Limit(requestRate), requestRate)

	var completedRequests int32
	var mu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				if err := limiter.Wait(ctx); err != nil {
					return // Context timeout
				}

				mu.Lock()
				completedRequests++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return int(completedRequests)
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
