package stats

import (
	"sync"
	"time"
)

// Result represents the result of crawling a single URL
type Result struct {
	URL         string        `json:"url"`
	Success     bool          `json:"success"`
	StatusCode  int           `json:"status_code,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	CacheStatus string        `json:"cache_status,omitempty"`
}

// Progress represents current crawling progress
type Progress struct {
	Processed         int           `json:"processed"`
	Total             int           `json:"total"`
	Percentage        float64       `json:"percentage"`
	SuccessRate       float64       `json:"success_rate"`
	AverageDuration   time.Duration `json:"average_duration"`
	ElapsedTime       time.Duration `json:"elapsed_time"`
	EstimatedTimeLeft time.Duration `json:"estimated_time_left"`
	RequestsPerSecond float64       `json:"requests_per_second"`
}

// FinalStats represents final crawling statistics
type FinalStats struct {
	TotalProcessed  int           `json:"total_processed"`
	TotalSuccess    int           `json:"total_success"`
	TotalErrors     int           `json:"total_errors"`
	SuccessRate     float64       `json:"success_rate"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	TotalDuration   time.Duration `json:"total_duration"`
}

// CacheStats represents cache verification statistics
type CacheStats struct {
	CacheHits    int           `json:"cache_hits"`
	CacheMisses  int           `json:"cache_misses"`
	CacheHitRate float64       `json:"cache_hit_rate"`
	WarmUpTime   time.Duration `json:"warm_up_time"`
	VerifyTime   time.Duration `json:"verify_time"`
}

// Stats handles all statistics tracking
type Stats struct {
	mu sync.RWMutex

	// General stats
	totalURLs     int
	processed     int
	successCount  int
	errorCount    int
	totalDuration time.Duration
	minDuration   time.Duration
	maxDuration   time.Duration
	startTime     time.Time

	// Cache verification stats
	warmUpResults []*Result
	cacheResults  []*Result
	warmUpStart   time.Time
	warmUpEnd     time.Time
	verifyStart   time.Time
	verifyEnd     time.Time
}

// New creates a new Stats instance
func New() *Stats {
	return &Stats{
		minDuration: time.Hour, // Initialize with a large value
	}
}

// SetTotalURLs sets the total number of URLs to process
func (s *Stats) SetTotalURLs(total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalURLs = total
	s.startTime = time.Now() // Start timing when we know the total
}

// AddResult adds a crawling result
func (s *Stats) AddResult(result *Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.processed++
	s.totalDuration += result.Duration

	if result.Success {
		s.successCount++
	} else {
		s.errorCount++
	}

	// Update min/max duration
	if result.Duration < s.minDuration {
		s.minDuration = result.Duration
	}
	if result.Duration > s.maxDuration {
		s.maxDuration = result.Duration
	}
}

// AddWarmUpResult adds a warm-up phase result
func (s *Stats) AddWarmUpResult(result *Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.warmUpStart.IsZero() {
		s.warmUpStart = time.Now()
	}

	s.warmUpResults = append(s.warmUpResults, result)
	s.processed++
}

// AddCacheResult adds a cache verification phase result
func (s *Stats) AddCacheResult(result *Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.verifyStart.IsZero() {
		s.verifyStart = time.Now()
	}

	s.cacheResults = append(s.cacheResults, result)
}

// GetProgress returns current progress information
func (s *Stats) GetProgress() Progress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var percentage float64
	if s.totalURLs > 0 {
		percentage = float64(s.processed) / float64(s.totalURLs) * 100
	}

	var successRate float64
	if s.processed > 0 {
		successRate = float64(s.successCount) / float64(s.processed) * 100
	}

	var avgDuration time.Duration
	if s.processed > 0 {
		avgDuration = s.totalDuration / time.Duration(s.processed)
	}

	// Calculate elapsed time and ETA
	elapsedTime := time.Since(s.startTime)
	var estimatedTimeLeft time.Duration
	var requestsPerSecond float64

	if s.processed > 0 && elapsedTime > 0 {
		// Calculate requests per second
		requestsPerSecond = float64(s.processed) / elapsedTime.Seconds()

		// Calculate ETA based on current processing rate
		remaining := s.totalURLs - s.processed
		if remaining > 0 && requestsPerSecond > 0 {
			estimatedTimeLeft = time.Duration(float64(remaining)/requestsPerSecond) * time.Second
		}
	}

	return Progress{
		Processed:         s.processed,
		Total:             s.totalURLs,
		Percentage:        percentage,
		SuccessRate:       successRate,
		AverageDuration:   avgDuration,
		ElapsedTime:       elapsedTime,
		EstimatedTimeLeft: estimatedTimeLeft,
		RequestsPerSecond: requestsPerSecond,
	}
}

// GetFinalStats returns final statistics
func (s *Stats) GetFinalStats() FinalStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var successRate float64
	if s.processed > 0 {
		successRate = float64(s.successCount) / float64(s.processed) * 100
	}

	var avgDuration time.Duration
	if s.processed > 0 {
		avgDuration = s.totalDuration / time.Duration(s.processed)
	}

	// Handle case where minDuration wasn't updated (all requests failed)
	minDuration := s.minDuration
	if s.processed == 0 || s.minDuration == time.Hour {
		minDuration = 0
	}

	return FinalStats{
		TotalProcessed:  s.processed,
		TotalSuccess:    s.successCount,
		TotalErrors:     s.errorCount,
		SuccessRate:     successRate,
		AverageDuration: avgDuration,
		MinDuration:     minDuration,
		MaxDuration:     s.maxDuration,
		TotalDuration:   s.totalDuration,
	}
}

// GetCacheStats returns cache verification statistics
func (s *Stats) GetCacheStats() CacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Mark end times if not set
	if !s.warmUpEnd.IsZero() && s.verifyStart.IsZero() {
		s.warmUpEnd = time.Now()
	}
	if !s.verifyStart.IsZero() && s.verifyEnd.IsZero() {
		s.verifyEnd = time.Now()
	}

	// Calculate cache hit/miss rates
	var cacheHits, cacheMisses int
	for _, result := range s.cacheResults {
		if result.CacheStatus != "" {
			if result.CacheStatus == "HIT" || result.CacheStatus == "hit" {
				cacheHits++
			} else {
				cacheMisses++
			}
		}
	}

	var cacheHitRate float64
	totalCacheChecks := cacheHits + cacheMisses
	if totalCacheChecks > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCacheChecks) * 100
	}

	// Calculate timing
	warmUpTime := time.Duration(0)
	if !s.warmUpStart.IsZero() && !s.warmUpEnd.IsZero() {
		warmUpTime = s.warmUpEnd.Sub(s.warmUpStart)
	}

	verifyTime := time.Duration(0)
	if !s.verifyStart.IsZero() && !s.verifyEnd.IsZero() {
		verifyTime = s.verifyEnd.Sub(s.verifyStart)
	}

	return CacheStats{
		CacheHits:    cacheHits,
		CacheMisses:  cacheMisses,
		CacheHitRate: cacheHitRate,
		WarmUpTime:   warmUpTime,
		VerifyTime:   verifyTime,
	}
}

// Reset resets all statistics
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalURLs = 0
	s.processed = 0
	s.successCount = 0
	s.errorCount = 0
	s.totalDuration = 0
	s.minDuration = time.Hour
	s.maxDuration = 0
	s.warmUpResults = nil
	s.cacheResults = nil
	s.warmUpStart = time.Time{}
	s.warmUpEnd = time.Time{}
	s.verifyStart = time.Time{}
	s.verifyEnd = time.Time{}
}
