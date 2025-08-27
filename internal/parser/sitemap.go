package parser

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Sitemap represents a sitemap structure
type Sitemap struct {
	XMLName xml.Name `xml:"sitemapindex"`
	URLs    []URL    `xml:"sitemap"`
}

// URLSet represents a URL set in a sitemap
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []URL    `xml:"url"`
}

// URL represents a URL entry in a sitemap
type URL struct {
	Loc        string    `xml:"loc"`
	LastMod    time.Time `xml:"lastmod,omitempty"`
	ChangeFreq string    `xml:"changefreq,omitempty"`
	Priority   float64   `xml:"priority,omitempty"`
}

// Parser handles parsing of various sitemap formats
type Parser struct {
	client *http.Client
}

// NewParser creates a new sitemap parser
func NewParser(timeout time.Duration) *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// ParseSitemap parses a sitemap and returns all URLs to crawl
func (p *Parser) ParseSitemap(sitemapURL string, headers map[string]string) ([]string, error) {
	urls, err := p.fetchAndParse(sitemapURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sitemap %s: %w", sitemapURL, err)
	}

	// Check if this is a sitemap index (contains other sitemaps)
	if len(urls) > 0 && p.isSitemapIndex(urls) {
		// For now, just return the sitemap URLs from the index
		// In a real scenario, you might want to recursively process them
		return urls, nil
	}

	return urls, nil
}

// fetchAndParse fetches and parses a sitemap
func (p *Parser) fetchAndParse(sitemapURL string, headers map[string]string) ([]string, error) {
	req, err := http.NewRequest("GET", sitemapURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default user agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "SitemapCrawler/1.0")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return p.parseXML(body)
}

// parseXML parses XML content and extracts URLs
func (p *Parser) parseXML(data []byte) ([]string, error) {
	// Try to parse as sitemap index first
	var sitemap Sitemap
	if err := xml.Unmarshal(data, &sitemap); err == nil && len(sitemap.URLs) > 0 {
		urls := make([]string, len(sitemap.URLs))
		for i, url := range sitemap.URLs {
			urls[i] = url.Loc
		}
		return urls, nil
	}

	// Try to parse as URL set
	var urlSet URLSet
	if err := xml.Unmarshal(data, &urlSet); err == nil && len(urlSet.URLs) > 0 {
		urls := make([]string, len(urlSet.URLs))
		for i, url := range urlSet.URLs {
			urls[i] = url.Loc
		}
		return urls, nil
	}

	// Try to parse as plain text (one URL per line)
	text := string(data)
	lines := strings.Split(text, "\n")
	var urls []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && (strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://")) {
			urls = append(urls, line)
		}
	}

	if len(urls) > 0 {
		return urls, nil
	}

	return nil, fmt.Errorf("unable to parse sitemap format")
}

// isSitemapIndex checks if the URLs are likely sitemap URLs
func (p *Parser) isSitemapIndex(urls []string) bool {
	for _, url := range urls {
		if strings.Contains(url, "sitemap") || strings.HasSuffix(url, ".xml") {
			return true
		}
	}
	return false
}

// ValidateURL checks if a URL is valid
func (p *Parser) ValidateURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Must have a scheme
	if parsedURL.Scheme == "" {
		return false
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	// Must have a host
	if parsedURL.Host == "" {
		return false
	}

	return true
}
