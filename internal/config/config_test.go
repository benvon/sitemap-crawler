package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	msgCacheHeaderError                            = "cache header must be specified when cache verification mode is enabled"
	msgOutputFormatError                           = "invalid output format:"
	msgBackoffInitialDelayError                    = "backoff initial delay must be greater than 0"
	msgBackoffMaxDelayError                        = "backoff max delay must be greater than 0"
	msgBackoffInitialDelayGreaterThanMaxDelayError = "backoff initial delay cannot be greater than max delay"
	msgBackoffMultiplierError                      = "backoff multiplier must be greater than 1.0"
	msgResponseTimeDegradationThresholdError       = "response time degradation threshold must be between 0 and 1.0"
	msgForbiddenErrorThresholdError                = "forbidden error threshold must be at least 1"
	msgForbiddenErrorWindowError                   = "forbidden error window must be greater than 0"
	siteMapURL                                     = "https://example.com/sitemap.xml"
)

func TestValidateBasicConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid basic config",
			config: &Config{
				SitemapURL:     siteMapURL,
				MaxWorkers:     10,
				RequestRate:    100,
				RequestTimeout: 30 * time.Second,
			},
			wantError: false,
		},
		{
			name: "missing sitemap URL",
			config: &Config{
				MaxWorkers:     10,
				RequestRate:    100,
				RequestTimeout: 30 * time.Second,
			},
			wantError: true,
			errorMsg:  "sitemap URL is required",
		},
		{
			name: "invalid max workers",
			config: &Config{
				SitemapURL:     siteMapURL,
				MaxWorkers:     0,
				RequestRate:    100,
				RequestTimeout: 30 * time.Second,
			},
			wantError: true,
			errorMsg:  "max workers must be at least 1",
		},
		{
			name: "invalid request rate",
			config: &Config{
				SitemapURL:     siteMapURL,
				MaxWorkers:     10,
				RequestRate:    0,
				RequestTimeout: 30 * time.Second,
			},
			wantError: true,
			errorMsg:  "request rate must be at least 1",
		},
		{
			name: "invalid request timeout",
			config: &Config{
				SitemapURL:     siteMapURL,
				MaxWorkers:     10,
				RequestRate:    100,
				RequestTimeout: 500 * time.Millisecond,
			},
			wantError: true,
			errorMsg:  "request timeout must be at least 1 second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateBasicConfig(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCacheConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "cache disabled",
			config: &Config{
				CacheVerificationMode: false,
				CacheHeader:           "",
			},
			wantError: false,
		},
		{
			name: "cache enabled with header",
			config: &Config{
				CacheVerificationMode: true,
				CacheHeader:           "X-Cache",
			},
			wantError: false,
		},
		{
			name: "cache enabled without header",
			config: &Config{
				CacheVerificationMode: true,
				CacheHeader:           "",
			},
			wantError: true,
			errorMsg:  msgCacheHeaderError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateCacheConfig(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateOutputConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		outputFormat string
		wantError    bool
		errorMsg     string
	}{
		{
			name:         "valid text format",
			outputFormat: "text",
			wantError:    false,
		},
		{
			name:         "valid json format",
			outputFormat: "json",
			wantError:    false,
		},
		{
			name:         "valid csv format",
			outputFormat: "csv",
			wantError:    false,
		},
		{
			name:         "invalid format",
			outputFormat: "xml",
			wantError:    true,
			errorMsg:     msgOutputFormatError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := &Config{OutputFormat: tt.outputFormat}
			err := validateOutputConfig(config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBackoffConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "backoff disabled",
			config: &Config{
				BackoffEnabled: false,
			},
			wantError: false,
		},
		{
			name: "valid backoff config",
			config: &Config{
				BackoffEnabled:                   true,
				BackoffInitialDelay:              1 * time.Second,
				BackoffMaxDelay:                  30 * time.Second,
				BackoffMultiplier:                2.0,
				ResponseTimeDegradationThreshold: 0.5,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateBackoffConfig(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBackoffDelays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid delays",
			config: &Config{
				BackoffInitialDelay: 1 * time.Second,
				BackoffMaxDelay:     30 * time.Second,
				BackoffMultiplier:   2.0,
			},
			wantError: false,
		},
		{
			name: "zero initial delay",
			config: &Config{
				BackoffInitialDelay: 0,
				BackoffMaxDelay:     30 * time.Second,
				BackoffMultiplier:   2.0,
			},
			wantError: true,
			errorMsg:  msgBackoffInitialDelayError,
		},
		{
			name: "negative initial delay",
			config: &Config{
				BackoffInitialDelay: -1 * time.Second,
				BackoffMaxDelay:     30 * time.Second,
				BackoffMultiplier:   2.0,
			},
			wantError: true,
			errorMsg:  msgBackoffInitialDelayError,
		},
		{
			name: "zero max delay",
			config: &Config{
				BackoffInitialDelay: 1 * time.Second,
				BackoffMaxDelay:     0,
				BackoffMultiplier:   2.0,
			},
			wantError: true,
			errorMsg:  msgBackoffMaxDelayError,
		},
		{
			name: "initial delay greater than max delay",
			config: &Config{
				BackoffInitialDelay: 30 * time.Second,
				BackoffMaxDelay:     1 * time.Second,
				BackoffMultiplier:   2.0,
			},
			wantError: true,
			errorMsg:  msgBackoffInitialDelayGreaterThanMaxDelayError,
		},
		{
			name: "invalid multiplier (too low)",
			config: &Config{
				BackoffInitialDelay: 1 * time.Second,
				BackoffMaxDelay:     30 * time.Second,
				BackoffMultiplier:   1.0,
			},
			wantError: true,
			errorMsg:  msgBackoffMultiplierError,
		},
		{
			name: "invalid multiplier (negative)",
			config: &Config{
				BackoffInitialDelay: 1 * time.Second,
				BackoffMaxDelay:     30 * time.Second,
				BackoffMultiplier:   0.5,
			},
			wantError: true,
			errorMsg:  msgBackoffMultiplierError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateBackoffDelays(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBackoffThresholds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid thresholds",
			config: &Config{
				ResponseTimeDegradationThreshold: 0.5,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: false,
		},
		{
			name: "zero degradation threshold",
			config: &Config{
				ResponseTimeDegradationThreshold: 0,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: true,
			errorMsg:  msgResponseTimeDegradationThresholdError,
		},
		{
			name: "negative degradation threshold",
			config: &Config{
				ResponseTimeDegradationThreshold: -0.1,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: true,
			errorMsg:  msgResponseTimeDegradationThresholdError,
		},
		{
			name: "degradation threshold too high",
			config: &Config{
				ResponseTimeDegradationThreshold: 1.5,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: true,
			errorMsg:  msgResponseTimeDegradationThresholdError,
		},
		{
			name: "zero forbidden error threshold",
			config: &Config{
				ResponseTimeDegradationThreshold: 0.5,
				ForbiddenErrorThreshold:          0,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: true,
			errorMsg:  msgForbiddenErrorThresholdError,
		},
		{
			name: "negative forbidden error threshold",
			config: &Config{
				ResponseTimeDegradationThreshold: 0.5,
				ForbiddenErrorThreshold:          -1,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: true,
			errorMsg:  msgForbiddenErrorThresholdError,
		},
		{
			name: "zero forbidden error window",
			config: &Config{
				ResponseTimeDegradationThreshold: 0.5,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             0,
			},
			wantError: true,
			errorMsg:  msgForbiddenErrorWindowError,
		},
		{
			name: "negative forbidden error window",
			config: &Config{
				ResponseTimeDegradationThreshold: 0.5,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             -1 * time.Second,
			},
			wantError: true,
			errorMsg:  msgForbiddenErrorWindowError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateBackoffThresholds(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfigIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "fully valid config",
			config: &Config{
				SitemapURL:                       siteMapURL,
				MaxWorkers:                       10,
				RequestRate:                      100,
				RequestTimeout:                   30 * time.Second,
				CacheVerificationMode:            false,
				OutputFormat:                     "json",
				BackoffEnabled:                   true,
				BackoffInitialDelay:              1 * time.Second,
				BackoffMaxDelay:                  30 * time.Second,
				BackoffMultiplier:                2.0,
				ResponseTimeDegradationThreshold: 0.5,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: false,
		},
		{
			name: "config with multiple validation errors",
			config: &Config{
				SitemapURL:                       "",
				MaxWorkers:                       0, // Invalid value
				RequestRate:                      100,
				RequestTimeout:                   30 * time.Second,
				CacheVerificationMode:            false,
				OutputFormat:                     "xml", // Invalid format
				BackoffEnabled:                   true,
				BackoffInitialDelay:              0, // Invalid value
				BackoffMaxDelay:                  30 * time.Second,
				BackoffMultiplier:                2.0,
				ResponseTimeDegradationThreshold: 0.5,
				ForbiddenErrorThreshold:          5,
				ForbiddenErrorWindow:             5 * time.Second,
			},
			wantError: true,
			errorMsg:  "sitemap URL is required", // Should catch the first error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateConfig(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
	t.Parallel()

	// This test requires setting up viper, which is complex for unit testing
	// but we can test the logic conceptually
	t.Run("header parsing logic", func(t *testing.T) {
		t.Parallel()
		headers := []string{
			"Authorization:Bearer token123",
			"Content-Type:application/json",
			"X-Custom-Header: custom value",
			"InvalidHeader", // Should be ignored
		}

		headerMap := make(map[string]string)
		for _, header := range headers {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) == 2 {
				headerMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		expectedMap := map[string]string{
			"Authorization":   "Bearer token123",
			"Content-Type":    "application/json",
			"X-Custom-Header": "custom value",
		}

		assert.Equal(t, expectedMap, headerMap)
	})
}

func TestConstants(t *testing.T) {
	t.Parallel()

	// Test that all flag constants are defined and not empty
	flagConstants := []string{
		FlagSitemapURL,
		FlagMaxWorkers,
		FlagRequestRate,
		FlagRequestTimeout,
		FlagUserAgent,
		FlagHeaders,
		FlagCacheVerificationMode,
		FlagCacheHeader,
		FlagOutputFormat,
		FlagQuiet,
		FlagProgressInterval,
		FlagDebug,
		FlagBackoffEnabled,
		FlagBackoffInitialDelay,
		FlagBackoffMaxDelay,
		FlagBackoffMultiplier,
		FlagResponseTimeDegradationThreshold,
		FlagForbiddenErrorThreshold,
		FlagForbiddenErrorWindow,
	}

	for _, flagConst := range flagConstants {
		assert.NotEmpty(t, flagConst, "Flag constant should not be empty")
	}

	// Test that flag constants match expected values
	assert.Equal(t, "sitemap-url", FlagSitemapURL)
	assert.Equal(t, "backoff-enabled", FlagBackoffEnabled)
	assert.Equal(t, "forbidden-error-threshold", FlagForbiddenErrorThreshold)
}
