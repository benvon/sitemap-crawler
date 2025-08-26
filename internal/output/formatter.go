package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/benvon/sitemap-crawler/internal/stats"
)

// Formatter handles output formatting for different formats
type Formatter struct {
	format string
}

// New creates a new formatter
func New(format string) *Formatter {
	return &Formatter{
		format: format,
	}
}

// FormatProgress formats progress information
func (f *Formatter) FormatProgress(progress *stats.Progress) string {
	switch f.format {
	case "json":
		return f.formatProgressJSON(progress)
	case "csv":
		return f.formatProgressCSV(progress)
	default:
		return f.formatProgressText(progress)
	}
}

// FormatFinalStats formats final statistics
func (f *Formatter) FormatFinalStats(finalStats *stats.FinalStats) string {
	switch f.format {
	case "json":
		return f.formatFinalStatsJSON(finalStats)
	case "csv":
		return f.formatFinalStatsCSV(finalStats)
	default:
		return f.formatFinalStatsText(finalStats)
	}
}

// FormatCacheStats formats cache verification statistics
func (f *Formatter) FormatCacheStats(cacheStats *stats.CacheStats) string {
	switch f.format {
	case "json":
		return f.formatCacheStatsJSON(cacheStats)
	case "csv":
		return f.formatCacheStatsCSV(cacheStats)
	default:
		return f.formatCacheStatsText(cacheStats)
	}
}

// WriteToFile writes formatted output to a file
func (f *Formatter) WriteToFile(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

// formatProgressText formats progress as text
func (f *Formatter) formatProgressText(progress *stats.Progress) string {
	return fmt.Sprintf(
		"Progress: %d/%d (%.1f%%) | Success Rate: %.1f%% | Avg Duration: %s",
		progress.Processed,
		progress.Total,
		progress.Percentage,
		progress.SuccessRate,
		progress.AverageDuration,
	)
}

// formatProgressJSON formats progress as JSON
func (f *Formatter) formatProgressJSON(progress *stats.Progress) string {
	data := map[string]interface{}{
		"timestamp":        time.Now().Format(time.RFC3339),
		"processed":        progress.Processed,
		"total":            progress.Total,
		"percentage":       progress.Percentage,
		"success_rate":     progress.SuccessRate,
		"average_duration": progress.AverageDuration.String(),
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}

// formatProgressCSV formats progress as CSV
func (f *Formatter) formatProgressCSV(progress *stats.Progress) string {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	if err := writer.Write([]string{
		"timestamp",
		"processed",
		"total",
		"percentage",
		"success_rate",
		"average_duration",
	}); err != nil {
		return ""
	}

	if err := writer.Write([]string{
		time.Now().Format(time.RFC3339),
		fmt.Sprintf("%d", progress.Processed),
		fmt.Sprintf("%d", progress.Total),
		fmt.Sprintf("%.1f", progress.Percentage),
		fmt.Sprintf("%.1f", progress.SuccessRate),
		progress.AverageDuration.String(),
	}); err != nil {
		return ""
	}

	writer.Flush()
	return builder.String()
}

// formatFinalStatsText formats final statistics as text
func (f *Formatter) formatFinalStatsText(finalStats *stats.FinalStats) string {
	return fmt.Sprintf(`
Final Statistics:
================
Total Processed:  %d
Total Success:    %d
Total Errors:     %d
Success Rate:     %.1f%%
Average Duration: %s
Min Duration:     %s
Max Duration:     %s
Total Duration:   %s
`,
		finalStats.TotalProcessed,
		finalStats.TotalSuccess,
		finalStats.TotalErrors,
		finalStats.SuccessRate,
		finalStats.AverageDuration,
		finalStats.MinDuration,
		finalStats.MaxDuration,
		finalStats.TotalDuration,
	)
}

// formatFinalStatsJSON formats final statistics as JSON
func (f *Formatter) formatFinalStatsJSON(finalStats *stats.FinalStats) string {
	data := map[string]interface{}{
		"timestamp":        time.Now().Format(time.RFC3339),
		"total_processed":  finalStats.TotalProcessed,
		"total_success":    finalStats.TotalSuccess,
		"total_errors":     finalStats.TotalErrors,
		"success_rate":     finalStats.SuccessRate,
		"average_duration": finalStats.AverageDuration.String(),
		"min_duration":     finalStats.MinDuration.String(),
		"max_duration":     finalStats.MaxDuration.String(),
		"total_duration":   finalStats.TotalDuration.String(),
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}

// formatFinalStatsCSV formats final statistics as CSV
func (f *Formatter) formatFinalStatsCSV(finalStats *stats.FinalStats) string {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	if err := writer.Write([]string{
		"timestamp",
		"total_processed",
		"total_success",
		"total_errors",
		"success_rate",
		"average_duration",
		"min_duration",
		"max_duration",
		"total_duration",
	}); err != nil {
		return ""
	}

	if err := writer.Write([]string{
		time.Now().Format(time.RFC3339),
		fmt.Sprintf("%d", finalStats.TotalProcessed),
		fmt.Sprintf("%d", finalStats.TotalSuccess),
		fmt.Sprintf("%d", finalStats.TotalErrors),
		fmt.Sprintf("%.1f", finalStats.SuccessRate),
		finalStats.AverageDuration.String(),
		finalStats.MinDuration.String(),
		finalStats.MaxDuration.String(),
		finalStats.TotalDuration.String(),
	}); err != nil {
		return ""
	}

	writer.Flush()
	return builder.String()
}

// formatCacheStatsText formats cache statistics as text
func (f *Formatter) formatCacheStatsText(cacheStats *stats.CacheStats) string {
	return fmt.Sprintf(`
Cache Verification Statistics:
============================
Cache Hits:       %d
Cache Misses:     %d
Cache Hit Rate:   %.1f%%
Warm Up Time:     %s
Verification Time: %s
`,
		cacheStats.CacheHits,
		cacheStats.CacheMisses,
		cacheStats.CacheHitRate,
		cacheStats.WarmUpTime,
		cacheStats.VerifyTime,
	)
}

// formatCacheStatsJSON formats cache statistics as JSON
func (f *Formatter) formatCacheStatsJSON(cacheStats *stats.CacheStats) string {
	data := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"cache_hits":     cacheStats.CacheHits,
		"cache_misses":   cacheStats.CacheMisses,
		"cache_hit_rate": cacheStats.CacheHitRate,
		"warm_up_time":   cacheStats.WarmUpTime.String(),
		"verify_time":    cacheStats.VerifyTime.String(),
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}

// formatCacheStatsCSV formats cache statistics as CSV
func (f *Formatter) formatCacheStatsCSV(cacheStats *stats.CacheStats) string {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	if err := writer.Write([]string{
		"timestamp",
		"cache_hits",
		"cache_misses",
		"cache_hit_rate",
		"warm_up_time",
		"verify_time",
	}); err != nil {
		return ""
	}

	if err := writer.Write([]string{
		time.Now().Format(time.RFC3339),
		fmt.Sprintf("%d", cacheStats.CacheHits),
		fmt.Sprintf("%d", cacheStats.CacheMisses),
		fmt.Sprintf("%.1f", cacheStats.CacheHitRate),
		cacheStats.WarmUpTime.String(),
		cacheStats.VerifyTime.String(),
	}); err != nil {
		return ""
	}

	writer.Flush()
	return builder.String()
}
