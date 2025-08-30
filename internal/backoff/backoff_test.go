package backoff

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func getTestConfig() Config {
	return Config{
		Enabled:                          true,
		InitialDelay:                     1 * time.Second,
		MaxDelay:                         30 * time.Second,
		Multiplier:                       2.0,
		ResponseTimeDegradationThreshold: 0.5,
		ForbiddenErrorThreshold:          5,
		ForbiddenErrorWindow:             5 * time.Second,
	}
}

func getDisabledTestConfig() Config {
	config := getTestConfig()
	config.Enabled = false
	return config
}

func getLowMaxDelayTestConfig() Config {
	config := getTestConfig()
	config.MaxDelay = 5 * time.Second // Low max delay for testing
	return config
}

func getLowThresholdTestConfig() Config {
	config := getTestConfig()
	config.ForbiddenErrorThreshold = 3 // Low threshold for testing
	return config
}

func getShortWindowTestConfig() Config {
	config := getTestConfig()
	config.ForbiddenErrorThreshold = 3                   // Low threshold for testing
	config.ForbiddenErrorWindow = 100 * time.Millisecond // Very short window for testing
	return config
}

func getShortWindow2TestConfig() Config {
	config := getTestConfig()
	config.ForbiddenErrorWindow = 100 * time.Millisecond // Short window
	return config
}

func getVeryLowThresholdTestConfig() Config {
	config := getTestConfig()
	config.ForbiddenErrorThreshold = 1 // Low threshold for easy testing
	return config
}

func getHighThresholdTestConfig() Config {
	config := getTestConfig()
	config.ResponseTimeDegradationThreshold = 0.8 // Higher threshold (80%) to make testing easier
	return config
}

func getHighForbiddenThresholdTestConfig() Config {
	config := getTestConfig()
	config.ForbiddenErrorThreshold = 10
	return config
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	manager := NewManager(logger, getTestConfig())

	assert.NotNil(t, manager)
	assert.True(t, manager.enabled)
	assert.Equal(t, 1*time.Second, manager.initialDelay)
	assert.Equal(t, 30*time.Second, manager.maxDelay)
	assert.Equal(t, 2.0, manager.multiplier)
	assert.Equal(t, 0.5, manager.responseTimeDegradationThreshold)
	assert.Equal(t, 5, manager.forbiddenErrorThreshold)
	assert.Equal(t, 5*time.Second, manager.forbiddenErrorWindow)
}

func TestShouldBackoff_ServerError(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getTestConfig())

	// Test 500 error triggers backoff
	shouldBackoff, delay, err := manager.ShouldBackoff(500, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, shouldBackoff)
	assert.Equal(t, 1*time.Second, delay)
	assert.True(t, manager.IsBackoffActive())

	// Test subsequent 500 error increases delay
	shouldBackoff, delay, err = manager.ShouldBackoff(500, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, shouldBackoff)
	assert.Equal(t, 2*time.Second, delay)

	// Test 502 error also triggers backoff
	shouldBackoff, delay, err = manager.ShouldBackoff(502, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, shouldBackoff)
	assert.Equal(t, 4*time.Second, delay)
}

func TestShouldBackoff_MaxDelay(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getLowMaxDelayTestConfig())

	// Trigger multiple server errors
	_, _, _ = manager.ShouldBackoff(500, 100*time.Millisecond)                    // 1s
	_, _, _ = manager.ShouldBackoff(500, 100*time.Millisecond)                    // 2s
	_, _, _ = manager.ShouldBackoff(500, 100*time.Millisecond)                    // 4s
	shouldBackoff, delay, err := manager.ShouldBackoff(500, 100*time.Millisecond) // Should cap at 5s

	assert.NoError(t, err)
	assert.True(t, shouldBackoff)
	assert.Equal(t, 5*time.Second, delay) // Should be capped at max delay
}

func TestShouldBackoff_ResetOnSuccess(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getTestConfig())

	// Trigger backoff
	shouldBackoff, _, err := manager.ShouldBackoff(500, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, shouldBackoff)
	assert.True(t, manager.IsBackoffActive())

	// Success should reset backoff
	shouldBackoff, _, err = manager.ShouldBackoff(200, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.False(t, shouldBackoff)
	assert.False(t, manager.IsBackoffActive())
}

func TestShouldBackoff_ForbiddenErrors(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getLowThresholdTestConfig())

	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.SetCancelFunc(cancel)

	// Add 403 errors below threshold
	_, _, err := manager.ShouldBackoff(403, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.False(t, manager.IsCancelled())

	_, _, err = manager.ShouldBackoff(403, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.False(t, manager.IsCancelled())

	// Third 403 error should trigger cancellation
	_, _, err = manager.ShouldBackoff(403, 100*time.Millisecond)
	assert.Error(t, err)
	assert.True(t, manager.IsCancelled())
	assert.Contains(t, err.Error(), "crawl cancelled")
}

func TestShouldBackoff_ForbiddenErrorsWindow(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getShortWindowTestConfig())

	// Add two 403 errors
	_, _, err := manager.ShouldBackoff(403, 100*time.Millisecond)
	assert.NoError(t, err)

	_, _, err = manager.ShouldBackoff(403, 100*time.Millisecond)
	assert.NoError(t, err)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Add third 403 error - should not trigger cancellation as previous errors are outside window
	_, _, err = manager.ShouldBackoff(403, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.False(t, manager.IsCancelled())
}

func TestShouldBackoff_ResponseTimeDegradation(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getTestConfig())

	// Establish baseline with fast responses
	for i := 0; i < 15; i++ {
		_, _, err := manager.ShouldBackoff(200, 100*time.Millisecond)
		assert.NoError(t, err)
	}

	// Ensure baseline is established
	stats := manager.GetStats()
	baselineTime, ok := stats["baseline_response_time"].(time.Duration)
	assert.True(t, ok)
	assert.Greater(t, baselineTime, time.Duration(0))

	// Add slow responses that exceed degradation threshold
	for i := 0; i < 10; i++ {
		shouldBackoff, delay, err := manager.ShouldBackoff(200, 200*time.Millisecond) // 100% slower
		if shouldBackoff {
			assert.NoError(t, err)
			assert.Equal(t, 1*time.Second, delay)
			assert.True(t, manager.IsBackoffActive())
			break
		}
		assert.NoError(t, err)
	}
}

func TestShouldBackoff_Disabled(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getDisabledTestConfig())

	// Server error should not trigger backoff when disabled
	shouldBackoff, delay, err := manager.ShouldBackoff(500, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.False(t, shouldBackoff)
	assert.Equal(t, time.Duration(0), delay)
	assert.False(t, manager.IsBackoffActive())
}

func TestGetStats(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getTestConfig())

	stats := manager.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "backoff_active")
	assert.Contains(t, stats, "current_delay")
	assert.Contains(t, stats, "baseline_response_time")
	assert.Contains(t, stats, "current_avg_response")
	assert.Contains(t, stats, "forbidden_errors_count")
	assert.Contains(t, stats, "cancelled")

	// Initially should not be active
	assert.False(t, stats["backoff_active"].(bool))
	assert.Equal(t, 0, stats["forbidden_errors_count"].(int))
	assert.False(t, stats["cancelled"].(bool))
}

func TestCleanOldForbiddenErrors(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getShortWindow2TestConfig())

	// Add some 403 errors
	_, _, _ = manager.ShouldBackoff(403, 100*time.Millisecond)
	_, _, _ = manager.ShouldBackoff(403, 100*time.Millisecond)

	stats := manager.GetStats()
	assert.Equal(t, 2, stats["forbidden_errors_count"].(int))

	// Wait for window to expire and add another error
	time.Sleep(150 * time.Millisecond)
	_, _, _ = manager.ShouldBackoff(403, 100*time.Millisecond)

	// Should only have 1 error (the recent one)
	stats = manager.GetStats()
	assert.Equal(t, 1, stats["forbidden_errors_count"].(int))
}

func TestSetCancelFunc(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getTestConfig())

	// Initially no cancel function
	assert.Nil(t, manager.cancelFunc)

	// Set cancel function
	_, cancel := context.WithCancel(context.Background())
	manager.SetCancelFunc(cancel)

	// Check that cancel function is set (can't easily test the function itself)
	assert.NotNil(t, manager.cancelFunc)
}

func TestShouldBackoff_EdgeCases(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getTestConfig())

	// Test different status codes that should not trigger backoff
	testCases := []int{100, 200, 201, 300, 301, 400, 401, 404, 499}

	for _, statusCode := range testCases {
		shouldBackoff, delay, err := manager.ShouldBackoff(statusCode, 100*time.Millisecond)
		assert.NoError(t, err)
		assert.False(t, shouldBackoff)
		assert.Equal(t, time.Duration(0), delay)
	}

	// Test all 50x status codes that should trigger backoff
	serverErrorCodes := []int{500, 501, 502, 503, 504, 505, 599}

	for _, statusCode := range serverErrorCodes {
		// Reset backoff state for each test
		manager.backoffActive = false
		manager.currentDelay = manager.initialDelay

		shouldBackoff, delay, err := manager.ShouldBackoff(statusCode, 100*time.Millisecond)
		assert.NoError(t, err)
		assert.True(t, shouldBackoff)
		assert.Equal(t, 1*time.Second, delay)
	}
}

func TestShouldBackoff_AlreadyCancelled(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getVeryLowThresholdTestConfig())

	// Trigger cancellation
	_, _, _ = manager.ShouldBackoff(403, 100*time.Millisecond)
	assert.True(t, manager.IsCancelled())

	// Subsequent calls should return error
	shouldBackoff, delay, err := manager.ShouldBackoff(200, 100*time.Millisecond)
	assert.Error(t, err)
	assert.False(t, shouldBackoff)
	assert.Equal(t, time.Duration(0), delay)
	assert.Contains(t, err.Error(), "crawl cancelled due to too many 403 errors")
}

func TestResponseTimeTracking_EdgeCases(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getHighThresholdTestConfig())

	// Test with zero duration responses
	for i := 0; i < 15; i++ {
		_, _, err := manager.ShouldBackoff(200, 0)
		assert.NoError(t, err)
	}

	stats := manager.GetStats()
	baseline, ok := stats["baseline_response_time"].(time.Duration)
	assert.True(t, ok)
	assert.Equal(t, time.Duration(0), baseline)

	// Test response time window overflow (more than 20 responses)
	for i := 0; i < 25; i++ {
		_, _, err := manager.ShouldBackoff(200, time.Duration(i+1)*time.Millisecond)
		assert.NoError(t, err)
	}

	// Should only track the last 20 responses
	assert.Len(t, manager.recentResponseTimes, 20)
}

func TestResetBackoff_WhenNotActive(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getTestConfig())

	// Call resetBackoff when backoff is not active (should be no-op)
	initialDelay := manager.currentDelay
	manager.resetBackoff()
	assert.False(t, manager.backoffActive)
	assert.Equal(t, initialDelay, manager.currentDelay)
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	manager := NewManager(logger, getHighForbiddenThresholdTestConfig())

	// Test concurrent access to ShouldBackoff
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			_, _, _ = manager.ShouldBackoff(200, 100*time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.GetStats()
			_ = manager.IsBackoffActive()
			_ = manager.IsCancelled()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should not panic and stats should be accessible
	stats := manager.GetStats()
	assert.NotNil(t, stats)
}
