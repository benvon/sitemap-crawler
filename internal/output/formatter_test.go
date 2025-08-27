package output

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benvon/sitemap-crawler/internal/stats"
)

func TestNew(t *testing.T) {
	t.Parallel()

	f := New("json")
	if f == nil {
		t.Fatal("Formatter should not be nil")
	}

	if f.format != "json" {
		t.Errorf("Expected format 'json', got '%s'", f.format)
	}
}

func TestFormatProgress(t *testing.T) {
	t.Parallel()

	progress := &stats.Progress{
		Processed:       5,
		Total:           10,
		Percentage:      50.0,
		SuccessRate:     80.0,
		AverageDuration: 150 * time.Millisecond,
	}

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "text format",
			format:   "text",
			expected: "Progress: 5/10 (50.0%) | Success Rate: 80.0% | Avg Duration: 150ms",
		},
		{
			name:     "json format",
			format:   "json",
			expected: `"processed": 5`,
		},
		{
			name:     "csv format",
			format:   "csv",
			expected: "timestamp,processed,total,percentage,success_rate,average_duration",
		},
		{
			name:     "unknown format defaults to text",
			format:   "unknown",
			expected: "Progress: 5/10 (50.0%) | Success Rate: 80.0% | Avg Duration: 150ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.format)
			result := f.FormatProgress(progress)

			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to contain '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFormatFinalStats(t *testing.T) {
	t.Parallel()

	finalStats := &stats.FinalStats{
		TotalProcessed:  10,
		TotalSuccess:    8,
		TotalErrors:     2,
		SuccessRate:     80.0,
		AverageDuration: 150 * time.Millisecond,
		MinDuration:     100 * time.Millisecond,
		MaxDuration:     200 * time.Millisecond,
		TotalDuration:   1500 * time.Millisecond,
	}

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "text format",
			format:   "text",
			expected: "Total Processed:  10",
		},
		{
			name:     "json format",
			format:   "json",
			expected: `"total_processed": 10`,
		},
		{
			name:     "csv format",
			format:   "csv",
			expected: "timestamp,total_processed,total_success,total_errors,success_rate,average_duration,min_duration,max_duration,total_duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.format)
			result := f.FormatFinalStats(finalStats)

			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to contain '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFormatCacheStats(t *testing.T) {
	t.Parallel()

	cacheStats := &stats.CacheStats{
		CacheHits:    6,
		CacheMisses:  4,
		CacheHitRate: 60.0,
		WarmUpTime:   500 * time.Millisecond,
		VerifyTime:   300 * time.Millisecond,
	}

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "text format",
			format:   "text",
			expected: "Cache Hits:       6",
		},
		{
			name:     "json format",
			format:   "json",
			expected: `"cache_hits": 6`,
		},
		{
			name:     "csv format",
			format:   "csv",
			expected: "timestamp,cache_hits,cache_misses,cache_hit_rate,warm_up_time,verify_time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.format)
			result := f.FormatCacheStats(cacheStats)

			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to contain '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFormatProgressText(t *testing.T) {
	t.Parallel()

	f := New("text")
	progress := &stats.Progress{
		Processed:       25,
		Total:           100,
		Percentage:      25.0,
		SuccessRate:     92.0,
		AverageDuration: 250 * time.Millisecond,
	}

	result := f.FormatProgress(progress)
	expected := "Progress: 25/100 (25.0%) | Success Rate: 92.0% | Avg Duration: 250ms"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatProgressJSON(t *testing.T) {
	t.Parallel()

	f := New("json")
	progress := &stats.Progress{
		Processed:       10,
		Total:           50,
		Percentage:      20.0,
		SuccessRate:     90.0,
		AverageDuration: 100 * time.Millisecond,
	}

	result := f.FormatProgress(progress)

	// Check that it's valid JSON
	if !strings.Contains(result, `"processed": 10`) {
		t.Errorf("Expected JSON to contain processed count, got '%s'", result)
	}

	if !strings.Contains(result, `"percentage": 20`) {
		t.Errorf("Expected JSON to contain percentage, got '%s'", result)
	}

	if !strings.Contains(result, `"average_duration": "100ms"`) {
		t.Errorf("Expected JSON to contain average duration, got '%s'", result)
	}
}

func TestFormatProgressCSV(t *testing.T) {
	t.Parallel()

	f := New("csv")
	progress := &stats.Progress{
		Processed:       15,
		Total:           75,
		Percentage:      20.0,
		SuccessRate:     85.0,
		AverageDuration: 175 * time.Millisecond,
	}

	result := f.FormatProgress(progress)

	// Check CSV structure
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 CSV lines, got %d", len(lines))
	}

	// Check header
	header := strings.Split(lines[0], ",")
	expectedHeaders := []string{"timestamp", "processed", "total", "percentage", "success_rate", "average_duration"}
	for i, expected := range expectedHeaders {
		if i >= len(header) || header[i] != expected {
			t.Errorf("Expected header[%d] to be '%s', got '%s'", i, expected, header[i])
		}
	}

	// Check data row
	data := strings.Split(lines[1], ",")
	if len(data) != len(expectedHeaders) {
		t.Errorf("Expected %d data columns, got %d", len(expectedHeaders), len(data))
	}
}

func TestWriteToFile(t *testing.T) {
	t.Parallel()

	f := New("text")
	testContent := "test content"
	testFile := "test_output.txt"

	// Clean up after test
	defer func() {
		if err := os.Remove(testFile); err != nil {
			t.Logf("Failed to remove test file: %v", err)
		}
	}()

	err := f.WriteToFile(testFile, testContent)
	if err != nil {
		t.Errorf("WriteToFile failed: %v", err)
	}

	// Verify file was written
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read test file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content '%s', got '%s'", testContent, string(content))
	}
}
