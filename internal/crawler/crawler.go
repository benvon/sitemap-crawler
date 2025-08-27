package crawler

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/benvon/sitemap-crawler/internal/config"
	"github.com/benvon/sitemap-crawler/internal/parser"
	"github.com/benvon/sitemap-crawler/internal/stats"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// Crawler handles the crawling process
type Crawler struct {
	config *config.Config
	logger *logrus.Logger
	parser *parser.Parser
	stats  *stats.Stats
	client *http.Client
}

// New creates a new crawler instance
func New(cfg *config.Config, logger *logrus.Logger) *Crawler {
	return &Crawler{
		config: cfg,
		logger: logger,
		parser: parser.NewParser(cfg.RequestTimeout),
		stats:  stats.New(),
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

// Run executes the crawling process
func (c *Crawler) Run() error {
	c.logger.Info("Starting sitemap crawler")
	c.logger.WithFields(logrus.Fields{
		"sitemap_url":  c.config.SitemapURL,
		"max_workers":  c.config.MaxWorkers,
		"request_rate": c.config.RequestRate,
		"cache_mode":   c.config.CacheVerificationMode,
	}).Info("Configuration loaded")

	// Parse sitemap to get URLs
	urls, err := c.parser.ParseSitemap(c.config.SitemapURL, c.config.Headers)
	if err != nil {
		return fmt.Errorf("failed to parse sitemap: %w", err)
	}

	c.logger.WithField("total_urls", len(urls)).Info("Sitemap parsed successfully")

	// Filter valid URLs
	validURLs := c.filterValidURLs(urls)
	c.logger.WithField("valid_urls", len(validURLs)).Info("URLs filtered")

	if len(validURLs) == 0 {
		return fmt.Errorf("no valid URLs found in sitemap")
	}

	// Initialize stats
	c.stats.SetTotalURLs(len(validURLs))

	// Run crawler
	if c.config.CacheVerificationMode {
		return c.runWithCacheVerification(validURLs)
	}

	return c.runStandardCrawl(validURLs)
}

// runStandardCrawl runs the standard crawling process
func (c *Crawler) runStandardCrawl(urls []string) error {
	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(c.config.RequestRate), c.config.RequestRate)

	// Create worker pool
	urlChan := make(chan string, len(urls))
	resultChan := make(chan *stats.Result, len(urls))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < c.config.MaxWorkers; i++ {
		wg.Add(1)
		go c.worker(i, urlChan, resultChan, limiter, &wg)
	}

	// Send URLs to workers
	go func() {
		defer close(urlChan)
		for _, url := range urls {
			urlChan <- url
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Start progress reporter
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !c.config.Quiet {
		go c.startProgressReporter(ctx)
	}

	// Process results and update stats
	for result := range resultChan {
		c.stats.AddResult(result)
	}

	c.printFinalStats()
	return nil
}

// runWithCacheVerification runs crawling with cache verification
func (c *Crawler) runWithCacheVerification(urls []string) error {
	c.logger.Info("Running in cache verification mode")

	// First pass: warm up cache
	c.logger.Info("Phase 1: Warming up cache")
	if err := c.warmUpCache(urls); err != nil {
		return fmt.Errorf("failed to warm up cache: %w", err)
	}

	// Second pass: verify cache
	c.logger.Info("Phase 2: Verifying cache")
	if err := c.verifyCache(urls); err != nil {
		return fmt.Errorf("failed to verify cache: %w", err)
	}

	c.printCacheStats()
	return nil
}

// warmUpCache performs initial requests to warm up the cache
func (c *Crawler) warmUpCache(urls []string) error {
	limiter := rate.NewLimiter(rate.Limit(c.config.RequestRate), c.config.RequestRate)

	urlChan := make(chan string, len(urls))
	resultChan := make(chan *stats.Result, len(urls))

	var wg sync.WaitGroup
	for i := 0; i < c.config.MaxWorkers; i++ {
		wg.Add(1)
		go c.worker(i, urlChan, resultChan, limiter, &wg)
	}

	go func() {
		defer close(urlChan)
		for _, url := range urls {
			urlChan <- url
		}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Start progress reporter for warm-up phase
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !c.config.Quiet {
		go c.startProgressReporter(ctx)
	}

	for result := range resultChan {
		c.stats.AddWarmUpResult(result)
	}

	return nil
}

// verifyCache performs second requests to check cache status
func (c *Crawler) verifyCache(urls []string) error {
	limiter := rate.NewLimiter(rate.Limit(c.config.RequestRate), c.config.RequestRate)

	urlChan := make(chan string, len(urls))
	resultChan := make(chan *stats.Result, len(urls))

	var wg sync.WaitGroup
	for i := 0; i < c.config.MaxWorkers; i++ {
		wg.Add(1)
		go c.worker(i, urlChan, resultChan, limiter, &wg)
	}

	go func() {
		defer close(urlChan)
		for _, url := range urls {
			urlChan <- url
		}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Start progress reporter for cache verification phase
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !c.config.Quiet {
		go c.startProgressReporter(ctx)
	}

	for result := range resultChan {
		c.stats.AddCacheResult(result)
	}

	return nil
}

// worker processes URLs from the channel
func (c *Crawler) worker(id int, urlChan <-chan string, resultChan chan<- *stats.Result, limiter *rate.Limiter, wg *sync.WaitGroup) {
	defer wg.Done()

	for url := range urlChan {
		// Wait for rate limiter
		if err := limiter.Wait(context.Background()); err != nil {
			c.logger.WithError(err).Error("Rate limiter error")
			continue
		}

		// Crawl URL
		result := c.crawlURL(url)
		resultChan <- result
	}
}

// crawlURL crawls a single URL and returns the result
func (c *Crawler) crawlURL(url string) *stats.Result {
	start := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &stats.Result{
			URL:      url,
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}
	}

	// Add custom headers
	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	// Set user agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.config.UserAgent)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return &stats.Result{
			URL:      url,
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.WithError(closeErr).Warn("Failed to close response body")
		}
	}()

	// Check cache status if in verification mode
	cacheStatus := ""
	if c.config.CacheVerificationMode {
		cacheStatus = resp.Header.Get(c.config.CacheHeader)
	}

	return &stats.Result{
		URL:         url,
		Success:     resp.StatusCode >= 200 && resp.StatusCode < 400,
		StatusCode:  resp.StatusCode,
		Duration:    time.Since(start),
		CacheStatus: cacheStatus,
	}
}

// filterValidURLs filters out invalid URLs
func (c *Crawler) filterValidURLs(urls []string) []string {
	var validURLs []string
	for _, url := range urls {
		if c.parser.ValidateURL(url) {
			validURLs = append(validURLs, url)
		}
	}
	return validURLs
}

// startProgressReporter starts a ticker-based progress reporter
func (c *Crawler) startProgressReporter(ctx context.Context) {
	ticker := time.NewTicker(c.config.ProgressInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.printProgress()
		}
	}
}

// printProgress prints current progress with enhanced information
func (c *Crawler) printProgress() {
	progress := c.stats.GetProgress()

	// Don't print if no progress yet
	if progress.Processed == 0 {
		return
	}

	// Format durations for better readability
	elapsedFormatted := c.formatDuration(progress.ElapsedTime)
	etaFormatted := c.formatDuration(progress.EstimatedTimeLeft)
	avgDurationFormatted := c.formatDuration(progress.AverageDuration)

	// Create a human-readable progress message
	message := fmt.Sprintf("Progress: %d/%d (%.1f%%) | Success Rate: %.1f%% | Speed: %.1f req/s | Elapsed: %s | ETA: %s | Avg Response: %s",
		progress.Processed,
		progress.Total,
		progress.Percentage,
		progress.SuccessRate,
		progress.RequestsPerSecond,
		elapsedFormatted,
		etaFormatted,
		avgDurationFormatted,
	)

	c.logger.Info(message)
}

// formatDuration formats a duration for human-readable display
func (c *Crawler) formatDuration(d time.Duration) string {
	if d == 0 {
		return "N/A"
	}

	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}

// printFinalStats prints final statistics
func (c *Crawler) printFinalStats() {
	stats := c.stats.GetFinalStats()

	c.logger.WithFields(logrus.Fields{
		"total_processed": stats.TotalProcessed,
		"total_success":   stats.TotalSuccess,
		"total_errors":    stats.TotalErrors,
		"success_rate":    fmt.Sprintf("%.1f%%", stats.SuccessRate),
		"avg_duration":    stats.AverageDuration,
		"min_duration":    stats.MinDuration,
		"max_duration":    stats.MaxDuration,
		"total_duration":  stats.TotalDuration,
	}).Info("Crawling completed")
}

// printCacheStats prints cache verification statistics
func (c *Crawler) printCacheStats() {
	cacheStats := c.stats.GetCacheStats()

	c.logger.WithFields(logrus.Fields{
		"cache_hits":     cacheStats.CacheHits,
		"cache_misses":   cacheStats.CacheMisses,
		"cache_hit_rate": fmt.Sprintf("%.1f%%", cacheStats.CacheHitRate),
		"warm_up_time":   cacheStats.WarmUpTime,
		"verify_time":    cacheStats.VerifyTime,
	}).Info("Cache verification completed")
}
