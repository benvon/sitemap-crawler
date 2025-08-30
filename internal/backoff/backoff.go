package backoff

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrorEvent represents an error event for tracking
type ErrorEvent struct {
	Timestamp  time.Time
	StatusCode int
	Duration   time.Duration
}

// Manager handles backoff logic and error tracking
type Manager struct {
	mu     sync.RWMutex
	logger *logrus.Logger

	// Configuration
	enabled                          bool
	initialDelay                     time.Duration
	maxDelay                         time.Duration
	multiplier                       float64
	responseTimeDegradationThreshold float64
	forbiddenErrorThreshold          int
	forbiddenErrorWindow             time.Duration

	// State
	currentDelay         time.Duration
	backoffActive        bool
	baselineResponseTime time.Duration
	recentResponseTimes  []time.Duration
	responseTimeWindow   int
	forbiddenErrors      []time.Time
	cancelled            bool
	cancelFunc           context.CancelFunc
}

// NewManager creates a new backoff manager
func NewManager(
	logger *logrus.Logger,
	enabled bool,
	initialDelay, maxDelay time.Duration,
	multiplier, responseTimeDegradationThreshold float64,
	forbiddenErrorThreshold int,
	forbiddenErrorWindow time.Duration,
) *Manager {
	return &Manager{
		logger:                           logger,
		enabled:                          enabled,
		initialDelay:                     initialDelay,
		maxDelay:                         maxDelay,
		multiplier:                       multiplier,
		responseTimeDegradationThreshold: responseTimeDegradationThreshold,
		forbiddenErrorThreshold:          forbiddenErrorThreshold,
		forbiddenErrorWindow:             forbiddenErrorWindow,
		currentDelay:                     initialDelay,
		responseTimeWindow:               20, // Track last 20 response times for baseline
		forbiddenErrors:                  make([]time.Time, 0),
	}
}

// SetCancelFunc sets the cancel function for the crawler context
func (m *Manager) SetCancelFunc(cancelFunc context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelFunc = cancelFunc
}

// ShouldBackoff determines if a backoff is needed based on the response
func (m *Manager) ShouldBackoff(statusCode int, duration time.Duration) (bool, time.Duration, error) {
	if !m.enabled {
		return false, 0, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for cancellation first
	if m.cancelled {
		return false, 0, fmt.Errorf("crawl cancelled due to too many 403 errors")
	}

	// Track 403 errors and check for cancellation threshold
	if statusCode == 403 {
		now := time.Now()
		m.forbiddenErrors = append(m.forbiddenErrors, now)

		// Clean old errors outside the window
		m.cleanOldForbiddenErrors(now)

		// Check if we've exceeded the threshold
		if len(m.forbiddenErrors) >= m.forbiddenErrorThreshold {
			m.cancelled = true
			m.logger.WithFields(logrus.Fields{
				"forbidden_errors": len(m.forbiddenErrors),
				"threshold":        m.forbiddenErrorThreshold,
				"window":           m.forbiddenErrorWindow,
			}).Error("Too many 403 errors detected, cancelling crawl")

			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			return false, 0, fmt.Errorf("crawl cancelled: %d 403 errors within %v window", len(m.forbiddenErrors), m.forbiddenErrorWindow)
		}
	}

	// Check for 50x errors
	if statusCode >= 500 && statusCode < 600 {
		m.logger.WithFields(logrus.Fields{
			"status_code":    statusCode,
			"current_delay":  m.currentDelay,
			"backoff_active": m.backoffActive,
		}).Warn("Server error detected, activating backoff")

		return m.activateBackoff(), m.currentDelay, nil
	}

	// Track response times for degradation detection
	m.trackResponseTime(duration)

	// Check for response time degradation
	if m.isResponseTimeDegraded() {
		m.logger.WithFields(logrus.Fields{
			"current_avg":           m.getCurrentAverageResponseTime(),
			"baseline":              m.baselineResponseTime,
			"degradation_threshold": m.responseTimeDegradationThreshold,
			"current_delay":         m.currentDelay,
			"backoff_active":        m.backoffActive,
		}).Warn("Response time degradation detected, activating backoff")

		return m.activateBackoff(), m.currentDelay, nil
	}

	// Reset backoff if we have a successful request and things seem normal
	if statusCode >= 200 && statusCode < 400 && m.backoffActive {
		m.resetBackoff()
	}

	return false, 0, nil
}

// activateBackoff activates or increases the backoff delay
func (m *Manager) activateBackoff() bool {
	if !m.backoffActive {
		m.backoffActive = true
		m.currentDelay = m.initialDelay
	} else {
		// Increase delay using exponential backoff
		newDelay := time.Duration(float64(m.currentDelay) * m.multiplier)
		if newDelay > m.maxDelay {
			m.currentDelay = m.maxDelay
		} else {
			m.currentDelay = newDelay
		}
	}
	return true
}

// resetBackoff resets the backoff state
func (m *Manager) resetBackoff() {
	if m.backoffActive {
		m.logger.WithField("previous_delay", m.currentDelay).Info("Resetting backoff, server appears healthy")
		m.backoffActive = false
		m.currentDelay = m.initialDelay
	}
}

// trackResponseTime adds a response time to the tracking window
func (m *Manager) trackResponseTime(duration time.Duration) {
	m.recentResponseTimes = append(m.recentResponseTimes, duration)

	// Keep only the last N response times
	if len(m.recentResponseTimes) > m.responseTimeWindow {
		m.recentResponseTimes = m.recentResponseTimes[1:]
	}

	// Set baseline if we have enough samples and no baseline yet
	if m.baselineResponseTime == 0 && len(m.recentResponseTimes) >= m.responseTimeWindow/2 {
		m.baselineResponseTime = m.getCurrentAverageResponseTime()
		m.logger.WithField("baseline_response_time", m.baselineResponseTime).Debug("Established baseline response time")
	}
}

// getCurrentAverageResponseTime calculates the current average response time
func (m *Manager) getCurrentAverageResponseTime() time.Duration {
	if len(m.recentResponseTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, duration := range m.recentResponseTimes {
		total += duration
	}
	return total / time.Duration(len(m.recentResponseTimes))
}

// isResponseTimeDegraded checks if response time has degraded significantly
func (m *Manager) isResponseTimeDegraded() bool {
	if m.baselineResponseTime == 0 || len(m.recentResponseTimes) < m.responseTimeWindow/2 {
		return false
	}

	currentAvg := m.getCurrentAverageResponseTime()
	degradationThreshold := time.Duration(float64(m.baselineResponseTime) * (1 + m.responseTimeDegradationThreshold))

	return currentAvg > degradationThreshold
}

// cleanOldForbiddenErrors removes forbidden errors outside the tracking window
func (m *Manager) cleanOldForbiddenErrors(now time.Time) {
	cutoff := now.Add(-m.forbiddenErrorWindow)

	// Find the first error within the window
	start := 0
	for i, errorTime := range m.forbiddenErrors {
		if errorTime.After(cutoff) {
			start = i
			break
		}
	}

	// If all errors are old, clear the slice
	if start == 0 && len(m.forbiddenErrors) > 0 && m.forbiddenErrors[len(m.forbiddenErrors)-1].Before(cutoff) {
		m.forbiddenErrors = m.forbiddenErrors[:0]
	} else if start > 0 {
		// Keep only errors within the window
		m.forbiddenErrors = m.forbiddenErrors[start:]
	}
}

// GetStats returns current backoff statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"backoff_active":         m.backoffActive,
		"current_delay":          m.currentDelay,
		"baseline_response_time": m.baselineResponseTime,
		"current_avg_response":   m.getCurrentAverageResponseTime(),
		"forbidden_errors_count": len(m.forbiddenErrors),
		"cancelled":              m.cancelled,
	}
}

// IsBackoffActive returns whether backoff is currently active
func (m *Manager) IsBackoffActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.backoffActive
}

// IsCancelled returns whether the crawl has been cancelled
func (m *Manager) IsCancelled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cancelled
}
