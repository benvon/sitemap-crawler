package stats

import (
	"fmt"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	s := New()
	if s == nil {
		t.Fatal("Stats should not be nil")
	}

	if s.minDuration != time.Hour {
		t.Errorf("Expected minDuration %v, got %v", time.Hour, s.minDuration)
	}
}

func TestSetTotalURLs(t *testing.T) {
	t.Parallel()

	s := New()
	s.SetTotalURLs(100)

	if s.totalURLs != 100 {
		t.Errorf("Expected totalURLs 100, got %d", s.totalURLs)
	}
}

func TestAddResult(t *testing.T) {
	t.Parallel()

	s := New()
	s.SetTotalURLs(2)

	// Add successful result
	result1 := &Result{
		URL:      "https://example.com/1",
		Success:  true,
		Duration: 100 * time.Millisecond,
	}
	s.AddResult(result1)

	// Add failed result
	result2 := &Result{
		URL:      "https://example.com/2",
		Success:  false,
		Duration: 200 * time.Millisecond,
	}
	s.AddResult(result2)

	if s.processed != 2 {
		t.Errorf("Expected processed 2, got %d", s.processed)
	}

	if s.successCount != 1 {
		t.Errorf("Expected successCount 1, got %d", s.successCount)
	}

	if s.errorCount != 1 {
		t.Errorf("Expected errorCount 1, got %d", s.errorCount)
	}

	if s.totalDuration != 300*time.Millisecond {
		t.Errorf("Expected totalDuration 300ms, got %v", s.totalDuration)
	}

	if s.minDuration != 100*time.Millisecond {
		t.Errorf("Expected minDuration 100ms, got %v", s.minDuration)
	}

	if s.maxDuration != 200*time.Millisecond {
		t.Errorf("Expected maxDuration 200ms, got %v", s.maxDuration)
	}
}

func TestGetProgress(t *testing.T) {
	t.Parallel()

	s := New()
	s.SetTotalURLs(10)

	// Add some results
	for i := 0; i < 5; i++ {
		s.AddResult(&Result{
			URL:      fmt.Sprintf("https://example.com/%d", i),
			Success:  i < 3, // 3 success, 2 failure
			Duration: time.Duration(i+1) * 100 * time.Millisecond,
		})
	}

	progress := s.GetProgress()

	if progress.Processed != 5 {
		t.Errorf("Expected Processed 5, got %d", progress.Processed)
	}

	if progress.Total != 10 {
		t.Errorf("Expected Total 10, got %d", progress.Total)
	}

	if progress.Percentage != 50.0 {
		t.Errorf("Expected Percentage 50.0, got %.1f", progress.Percentage)
	}

	if progress.SuccessRate != 60.0 {
		t.Errorf("Expected SuccessRate 60.0, got %.1f", progress.SuccessRate)
	}

	expectedAvgDuration := 300 * time.Millisecond // (100+200+300+400+500)/5
	if progress.AverageDuration != expectedAvgDuration {
		t.Errorf("Expected AverageDuration %v, got %v", expectedAvgDuration, progress.AverageDuration)
	}
}

func TestGetFinalStats(t *testing.T) {
	t.Parallel()

	s := New()
	s.SetTotalURLs(5)

	// Add results
	for i := 0; i < 5; i++ {
		s.AddResult(&Result{
			URL:      fmt.Sprintf("https://example.com/%d", i),
			Success:  i < 3, // 3 success, 2 failure
			Duration: time.Duration(i+1) * 100 * time.Millisecond,
		})
	}

	finalStats := s.GetFinalStats()

	if finalStats.TotalProcessed != 5 {
		t.Errorf("Expected TotalProcessed 5, got %d", finalStats.TotalProcessed)
	}

	if finalStats.TotalSuccess != 3 {
		t.Errorf("Expected TotalSuccess 3, got %d", finalStats.TotalSuccess)
	}

	if finalStats.TotalErrors != 2 {
		t.Errorf("Expected TotalErrors 2, got %d", finalStats.TotalErrors)
	}

	if finalStats.SuccessRate != 60.0 {
		t.Errorf("Expected SuccessRate 60.0, got %.1f", finalStats.SuccessRate)
	}

	expectedAvgDuration := 300 * time.Millisecond
	if finalStats.AverageDuration != expectedAvgDuration {
		t.Errorf("Expected AverageDuration %v, got %v", expectedAvgDuration, finalStats.AverageDuration)
	}

	if finalStats.MinDuration != 100*time.Millisecond {
		t.Errorf("Expected MinDuration 100ms, got %v", finalStats.MinDuration)
	}

	if finalStats.MaxDuration != 500*time.Millisecond {
		t.Errorf("Expected MaxDuration 500ms, got %v", finalStats.MaxDuration)
	}

	if finalStats.TotalDuration != 1500*time.Millisecond {
		t.Errorf("Expected TotalDuration 1500ms, got %v", finalStats.TotalDuration)
	}
}

func TestCacheVerification(t *testing.T) {
	t.Parallel()

	s := New()
	s.SetTotalURLs(4)

	// Add warm-up results
	s.AddWarmUpResult(&Result{
		URL:         "https://example.com/1",
		Success:     true,
		Duration:    100 * time.Millisecond,
		CacheStatus: "",
	})
	s.AddWarmUpResult(&Result{
		URL:         "https://example.com/2",
		Success:     true,
		Duration:    200 * time.Millisecond,
		CacheStatus: "",
	})

	// Add cache verification results
	s.AddCacheResult(&Result{
		URL:         "https://example.com/1",
		Success:     true,
		Duration:    50 * time.Millisecond,
		CacheStatus: "HIT",
	})
	s.AddCacheResult(&Result{
		URL:         "https://example.com/2",
		Success:     true,
		Duration:    150 * time.Millisecond,
		CacheStatus: "MISS",
	})

	cacheStats := s.GetCacheStats()

	if cacheStats.CacheHits != 1 {
		t.Errorf("Expected CacheHits 1, got %d", cacheStats.CacheHits)
	}

	if cacheStats.CacheMisses != 1 {
		t.Errorf("Expected CacheMisses 1, got %d", cacheStats.CacheMisses)
	}

	if cacheStats.CacheHitRate != 50.0 {
		t.Errorf("Expected CacheHitRate 50.0, got %.1f", cacheStats.CacheHitRate)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()

	s := New()
	s.SetTotalURLs(10)
	s.AddResult(&Result{
		URL:      "https://example.com",
		Success:  true,
		Duration: 100 * time.Millisecond,
	})

	s.Reset()

	if s.totalURLs != 0 {
		t.Errorf("Expected totalURLs 0 after reset, got %d", s.totalURLs)
	}

	if s.processed != 0 {
		t.Errorf("Expected processed 0 after reset, got %d", s.processed)
	}

	if s.successCount != 0 {
		t.Errorf("Expected successCount 0 after reset, got %d", s.successCount)
	}

	if s.errorCount != 0 {
		t.Errorf("Expected errorCount 0 after reset, got %d", s.errorCount)
	}

	if s.totalDuration != 0 {
		t.Errorf("Expected totalDuration 0 after reset, got %v", s.totalDuration)
	}
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	s := New()

	// Test with no URLs
	progress := s.GetProgress()
	if progress.Percentage != 0 {
		t.Errorf("Expected Percentage 0 for no URLs, got %.1f", progress.Percentage)
	}

	if progress.SuccessRate != 0 {
		t.Errorf("Expected SuccessRate 0 for no URLs, got %.1f", progress.SuccessRate)
	}

	// Test with no results
	finalStats := s.GetFinalStats()
	if finalStats.SuccessRate != 0 {
		t.Errorf("Expected SuccessRate 0 for no results, got %.1f", finalStats.SuccessRate)
	}

	if finalStats.MinDuration != 0 {
		t.Errorf("Expected MinDuration 0 for no results, got %v", finalStats.MinDuration)
	}
}
