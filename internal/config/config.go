package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Flag name constants to avoid duplication
const (
	FlagSitemapURL                       = "sitemap-url"
	FlagMaxWorkers                       = "max-workers"
	FlagRequestRate                      = "request-rate"
	FlagRequestTimeout                   = "request-timeout"
	FlagUserAgent                        = "user-agent"
	FlagHeaders                          = "headers"
	FlagCacheVerificationMode            = "cache-verification-mode"
	FlagCacheHeader                      = "cache-header"
	FlagOutputFormat                     = "output-format"
	FlagQuiet                            = "quiet"
	FlagProgressInterval                 = "progress-interval"
	FlagDebug                            = "debug"
	FlagBackoffEnabled                   = "backoff-enabled"
	FlagBackoffInitialDelay              = "backoff-initial-delay"
	FlagBackoffMaxDelay                  = "backoff-max-delay"
	FlagBackoffMultiplier                = "backoff-multiplier"
	FlagResponseTimeDegradationThreshold = "response-time-degradation-threshold"
	FlagForbiddenErrorThreshold          = "forbidden-error-threshold"
	FlagForbiddenErrorWindow             = "forbidden-error-window"
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

	// Backoff configuration
	BackoffEnabled                   bool          `mapstructure:"backoff-enabled"`
	BackoffInitialDelay              time.Duration `mapstructure:"backoff-initial-delay"`
	BackoffMaxDelay                  time.Duration `mapstructure:"backoff-max-delay"`
	BackoffMultiplier                float64       `mapstructure:"backoff-multiplier"`
	ResponseTimeDegradationThreshold float64       `mapstructure:"response-time-degradation-threshold"`
	ForbiddenErrorThreshold          int           `mapstructure:"forbidden-error-threshold"`
	ForbiddenErrorWindow             time.Duration `mapstructure:"forbidden-error-window"`
}

// Load loads configuration from command line flags and environment variables
func Load() (*Config, error) {
	cmd := createCommand()

	if err := addFlags(cmd); err != nil {
		return nil, fmt.Errorf("failed to add flags: %w", err)
	}

	if err := markRequiredFlags(cmd); err != nil {
		return nil, fmt.Errorf("failed to mark required flags: %w", err)
	}

	if err := cmd.Execute(); err != nil {
		return nil, fmt.Errorf("failed to parse command line: %w", err)
	}

	if err := bindFlags(cmd); err != nil {
		return nil, fmt.Errorf("failed to bind flags: %w", err)
	}

	if err := parseHeaders(); err != nil {
		return nil, fmt.Errorf("failed to parse headers: %w", err)
	}

	cfg, err := createConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return cfg, nil
}

// createCommand creates the cobra command with basic configuration
func createCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sitemap-crawler",
		Short: "A configurable sitemap crawling tool",
		Long: `A sitemap crawling tool that can interpret common sitemap formats,
including multi-stage and multi-file sitemaps. Features configurable request rates,
parallel workers, custom headers, and cache verification mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // We'll handle execution in main
		},
	}
}

// addFlags adds all command line flags to the command
func addFlags(cmd *cobra.Command) error {
	addBasicFlags(cmd)
	addCacheFlags(cmd)
	addOutputFlags(cmd)
	addBackoffFlags(cmd)
	return nil
}

// addBasicFlags adds basic crawler configuration flags
func addBasicFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagSitemapURL, "", "URL of the sitemap to crawl (required)")
	cmd.Flags().Int(FlagMaxWorkers, 10, "Maximum number of parallel workers")
	cmd.Flags().Int(FlagRequestRate, 100, "Maximum requests per second")
	cmd.Flags().Duration(FlagRequestTimeout, 30*time.Second, "Request timeout")
	cmd.Flags().String(FlagUserAgent, "SitemapCrawler/1.0", "User agent string")
	cmd.Flags().StringSlice(FlagHeaders, []string{}, "Custom headers in format 'Key:Value'")
}

// addCacheFlags adds cache verification flags
func addCacheFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(FlagCacheVerificationMode, false, "Enable cache verification mode")
	cmd.Flags().String(FlagCacheHeader, "X-Cache", "Header to check for cache status")
}

// addOutputFlags adds output configuration flags
func addOutputFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagOutputFormat, "text", "Output format (text, json, csv)")
	cmd.Flags().Bool(FlagQuiet, false, "Suppress progress output")
	cmd.Flags().Duration(FlagProgressInterval, 5*time.Second, "Progress report interval")
	cmd.Flags().Bool(FlagDebug, false, "Enable debug logging")
}

// addBackoffFlags adds backoff configuration flags
func addBackoffFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(FlagBackoffEnabled, true, "Enable backoff on server errors and response time degradation")
	cmd.Flags().Duration(FlagBackoffInitialDelay, 1*time.Second, "Initial backoff delay")
	cmd.Flags().Duration(FlagBackoffMaxDelay, 30*time.Second, "Maximum backoff delay")
	cmd.Flags().Float64(FlagBackoffMultiplier, 2.0, "Backoff delay multiplier")
	cmd.Flags().Float64(FlagResponseTimeDegradationThreshold, 0.5, "Response time degradation threshold (0.5 = 50% slower)")
	cmd.Flags().Int(FlagForbiddenErrorThreshold, 5, "Number of 403 errors within window to cancel crawl")
	cmd.Flags().Duration(FlagForbiddenErrorWindow, 5*time.Second, "Time window for 403 error tracking")
}

// markRequiredFlags marks flags that are required
func markRequiredFlags(cmd *cobra.Command) error {
	return cmd.MarkFlagRequired(FlagSitemapURL)
}

// bindFlags binds all flags to viper
func bindFlags(cmd *cobra.Command) error {
	flagNames := []string{
		FlagSitemapURL, FlagMaxWorkers, FlagRequestRate, FlagRequestTimeout, FlagUserAgent,
		FlagCacheVerificationMode, FlagCacheHeader, FlagOutputFormat, FlagQuiet,
		FlagProgressInterval, FlagDebug, FlagBackoffEnabled, FlagBackoffInitialDelay,
		FlagBackoffMaxDelay, FlagBackoffMultiplier, FlagResponseTimeDegradationThreshold,
		FlagForbiddenErrorThreshold, FlagForbiddenErrorWindow,
	}

	for _, flagName := range flagNames {
		if err := viper.BindPFlag(flagName, cmd.Flags().Lookup(flagName)); err != nil {
			return fmt.Errorf("failed to bind %s flag: %w", flagName, err)
		}
	}

	return nil
}

// parseHeaders parses the headers flag and sets up the header map
func parseHeaders() error {
	headers := viper.GetStringSlice(FlagHeaders)
	headerMap := make(map[string]string)
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			headerMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	viper.Set(FlagHeaders, headerMap)
	return nil
}

// createConfig creates and validates the final configuration
func createConfig() (*Config, error) {
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
	if err := validateBasicConfig(cfg); err != nil {
		return err
	}

	if err := validateCacheConfig(cfg); err != nil {
		return err
	}

	if err := validateOutputConfig(cfg); err != nil {
		return err
	}

	if err := validateBackoffConfig(cfg); err != nil {
		return err
	}

	return nil
}

// validateBasicConfig validates basic crawler configuration
func validateBasicConfig(cfg *Config) error {
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

	return nil
}

// validateCacheConfig validates cache verification configuration
func validateCacheConfig(cfg *Config) error {
	if cfg.CacheVerificationMode && cfg.CacheHeader == "" {
		return fmt.Errorf("cache header must be specified when cache verification mode is enabled")
	}

	return nil
}

// validateOutputConfig validates output configuration
func validateOutputConfig(cfg *Config) error {
	validFormats := map[string]bool{"text": true, "json": true, "csv": true}
	if !validFormats[cfg.OutputFormat] {
		return fmt.Errorf("invalid output format: %s (valid: text, json, csv)", cfg.OutputFormat)
	}

	return nil
}

// validateBackoffConfig validates backoff configuration
func validateBackoffConfig(cfg *Config) error {
	if !cfg.BackoffEnabled {
		return nil
	}

	if err := validateBackoffDelays(cfg); err != nil {
		return err
	}

	if err := validateBackoffThresholds(cfg); err != nil {
		return err
	}

	return nil
}

// validateBackoffDelays validates backoff delay configuration
func validateBackoffDelays(cfg *Config) error {
	if cfg.BackoffInitialDelay <= 0 {
		return fmt.Errorf("backoff initial delay must be greater than 0")
	}

	if cfg.BackoffMaxDelay <= 0 {
		return fmt.Errorf("backoff max delay must be greater than 0")
	}

	if cfg.BackoffInitialDelay > cfg.BackoffMaxDelay {
		return fmt.Errorf("backoff initial delay cannot be greater than max delay")
	}

	if cfg.BackoffMultiplier <= 1.0 {
		return fmt.Errorf("backoff multiplier must be greater than 1.0")
	}

	return nil
}

// validateBackoffThresholds validates backoff threshold configuration
func validateBackoffThresholds(cfg *Config) error {
	if cfg.ResponseTimeDegradationThreshold <= 0 || cfg.ResponseTimeDegradationThreshold > 1.0 {
		return fmt.Errorf("response time degradation threshold must be between 0 and 1.0")
	}

	if cfg.ForbiddenErrorThreshold < 1 {
		return fmt.Errorf("forbidden error threshold must be at least 1")
	}

	if cfg.ForbiddenErrorWindow <= 0 {
		return fmt.Errorf("forbidden error window must be greater than 0")
	}

	return nil
}
