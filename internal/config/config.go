package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all configuration for the sitemap crawler
type Config struct {
	// Sitemap configuration
	SitemapURL string `mapstructure:"sitemap-url"`

	// Crawling configuration
	MaxWorkers     int           `mapstructure:"max-workers"`
	RequestRate    int           `mapstructure:"request-rate"`
	RequestTimeout time.Duration `mapstructure:"request-timeout"`
	UserAgent      string        `mapstructure:"user-agent"`

	// Headers configuration
	Headers map[string]string `mapstructure:"headers"`

	// Cache verification mode
	CacheVerificationMode bool   `mapstructure:"cache-verification-mode"`
	CacheHeader           string `mapstructure:"cache-header"`

	// Output configuration
	OutputFormat     string        `mapstructure:"output-format"`
	Quiet            bool          `mapstructure:"quiet"`
	ProgressInterval time.Duration `mapstructure:"progress-interval"`

	// Debug mode
	Debug bool `mapstructure:"debug"`
}

// Load loads configuration from command line flags and environment variables
func Load() (*Config, error) {
	cmd := &cobra.Command{
		Use:   "sitemap-crawler",
		Short: "A configurable sitemap crawling tool",
		Long: `A sitemap crawling tool that can interpret common sitemap formats,
including multi-stage and multi-file sitemaps. Features configurable request rates,
parallel workers, custom headers, and cache verification mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // We'll handle execution in main
		},
	}

	// Add flags
	cmd.Flags().String("sitemap-url", "", "URL of the sitemap to crawl (required)")
	cmd.Flags().Int("max-workers", 10, "Maximum number of parallel workers")
	cmd.Flags().Int("request-rate", 100, "Maximum requests per second")
	cmd.Flags().Duration("request-timeout", 30*time.Second, "Request timeout")
	cmd.Flags().String("user-agent", "SitemapCrawler/1.0", "User agent string")
	cmd.Flags().StringSlice("headers", []string{}, "Custom headers in format 'Key:Value'")
	cmd.Flags().Bool("cache-verification-mode", false, "Enable cache verification mode")
	cmd.Flags().String("cache-header", "X-Cache", "Header to check for cache status")
	cmd.Flags().String("output-format", "text", "Output format (text, json, csv)")
	cmd.Flags().Bool("quiet", false, "Suppress progress output")
	cmd.Flags().Duration("progress-interval", 5*time.Second, "Progress report interval")
	cmd.Flags().Bool("debug", false, "Enable debug logging")

	// Mark required flags
	if err := cmd.MarkFlagRequired("sitemap-url"); err != nil {
		return nil, fmt.Errorf("failed to mark sitemap-url as required: %w", err)
	}

	// Parse command line
	if err := cmd.Execute(); err != nil {
		return nil, fmt.Errorf("failed to parse command line: %w", err)
	}

	// Bind flags to viper
	if err := viper.BindPFlag("sitemap-url", cmd.Flags().Lookup("sitemap-url")); err != nil {
		return nil, fmt.Errorf("failed to bind sitemap-url flag: %w", err)
	}
	if err := viper.BindPFlag("max-workers", cmd.Flags().Lookup("max-workers")); err != nil {
		return nil, fmt.Errorf("failed to bind max-workers flag: %w", err)
	}
	if err := viper.BindPFlag("request-rate", cmd.Flags().Lookup("request-rate")); err != nil {
		return nil, fmt.Errorf("failed to bind request-rate flag: %w", err)
	}
	if err := viper.BindPFlag("request-timeout", cmd.Flags().Lookup("request-timeout")); err != nil {
		return nil, fmt.Errorf("failed to bind request-timeout flag: %w", err)
	}
	if err := viper.BindPFlag("user-agent", cmd.Flags().Lookup("user-agent")); err != nil {
		return nil, fmt.Errorf("failed to bind user-agent flag: %w", err)
	}
	if err := viper.BindPFlag("cache-verification-mode", cmd.Flags().Lookup("cache-verification-mode")); err != nil {
		return nil, fmt.Errorf("failed to bind cache-verification-mode flag: %w", err)
	}
	if err := viper.BindPFlag("cache-header", cmd.Flags().Lookup("cache-header")); err != nil {
		return nil, fmt.Errorf("failed to bind cache-header flag: %w", err)
	}
	if err := viper.BindPFlag("output-format", cmd.Flags().Lookup("output-format")); err != nil {
		return nil, fmt.Errorf("failed to bind output-format flag: %w", err)
	}
	if err := viper.BindPFlag("quiet", cmd.Flags().Lookup("quiet")); err != nil {
		return nil, fmt.Errorf("failed to bind quiet flag: %w", err)
	}
	if err := viper.BindPFlag("progress-interval", cmd.Flags().Lookup("progress-interval")); err != nil {
		return nil, fmt.Errorf("failed to bind progress-interval flag: %w", err)
	}
	if err := viper.BindPFlag("debug", cmd.Flags().Lookup("debug")); err != nil {
		return nil, fmt.Errorf("failed to bind debug flag: %w", err)
	}

	// Parse headers
	headers := viper.GetStringSlice("headers")
	headerMap := make(map[string]string)
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			headerMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	viper.Set("headers", headerMap)

	// Set environment variable prefix
	viper.SetEnvPrefix("SITEMAP_CRAWLER")
	viper.AutomaticEnv()

	// Create config struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	if cfg.SitemapURL == "" {
		return fmt.Errorf("sitemap URL is required")
	}

	if cfg.MaxWorkers < 1 {
		return fmt.Errorf("max workers must be at least 1")
	}

	if cfg.RequestRate < 1 {
		return fmt.Errorf("request rate must be at least 1")
	}

	if cfg.RequestTimeout < time.Second {
		return fmt.Errorf("request timeout must be at least 1 second")
	}

	if cfg.CacheVerificationMode && cfg.CacheHeader == "" {
		return fmt.Errorf("cache header must be specified when cache verification mode is enabled")
	}

	validFormats := map[string]bool{"text": true, "json": true, "csv": true}
	if !validFormats[cfg.OutputFormat] {
		return fmt.Errorf("invalid output format: %s (valid: text, json, csv)", cfg.OutputFormat)
	}

	return nil
}
