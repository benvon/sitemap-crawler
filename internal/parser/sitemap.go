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

const (
	defaultUserAgent = "SitemapCrawler/1.0"
	maxSitemapBytes  = 50 * 1024 * 1024
	maxSitemapDepth  = 10
)

type parsedSitemap struct {
	urls    []string
	isIndex bool
}

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
	client    *http.Client
	userAgent string
}

// NewParser creates a new sitemap parser
func NewParser(timeout time.Duration) *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: timeout,
		},
		userAgent: defaultUserAgent,
	}
}

// SetUserAgent sets the User-Agent header used for sitemap fetches.
func (p *Parser) SetUserAgent(userAgent string) {
	if userAgent == "" {
		p.userAgent = defaultUserAgent
		return
	}
	p.userAgent = userAgent
}

// ParseSitemap parses a sitemap and returns all URLs to crawl
func (p *Parser) ParseSitemap(sitemapURL string, headers map[string]string) ([]string, error) {
	seenSitemaps := make(map[string]bool)
	seenURLs := make(map[string]bool)
	urls, err := p.parseSitemapRecursive(sitemapURL, headers, 0, seenSitemaps, seenURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sitemap %s: %w", sitemapURL, err)
	}

	return urls, nil
}

func (p *Parser) parseSitemapRecursive(sitemapURL string, headers map[string]string, depth int, seenSitemaps map[string]bool, seenURLs map[string]bool) ([]string, error) {
	if depth > maxSitemapDepth {
		return nil, fmt.Errorf("maximum sitemap depth exceeded")
	}
	if seenSitemaps[sitemapURL] {
		return nil, nil
	}
	seenSitemaps[sitemapURL] = true

	parsed, err := p.fetchAndParse(sitemapURL, headers)
	if err != nil {
		return nil, err
	}

	if !parsed.isIndex {
		return addUniqueURLs(nil, parsed.urls, seenURLs), nil
	}

	var urls []string
	for _, childSitemap := range parsed.urls {
		childURLs, err := p.parseSitemapRecursive(childSitemap, headers, depth+1, seenSitemaps, seenURLs)
		if err != nil {
			return nil, fmt.Errorf("failed to parse child sitemap %s: %w", childSitemap, err)
		}
		urls = append(urls, childURLs...)
	}

	return urls, nil
}

// fetchAndParse fetches and parses a sitemap
func (p *Parser) fetchAndParse(sitemapURL string, headers map[string]string) (parsedSitemap, error) {
	req, err := http.NewRequest("GET", sitemapURL, nil)
	if err != nil {
		return parsedSitemap{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default user agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", p.userAgent)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return parsedSitemap{}, fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return parsedSitemap{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := readLimited(resp.Body, maxSitemapBytes)
	if err != nil {
		return parsedSitemap{}, fmt.Errorf("failed to read response body: %w", err)
	}

	return p.parseSitemapContent(body)
}

func readLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(reader, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("sitemap exceeds maximum size of %d bytes", maxBytes)
	}
	return body, nil
}

// parseXML parses XML content and extracts URLs
func (p *Parser) parseXML(data []byte) ([]string, error) {
	parsed, err := p.parseSitemapContent(data)
	if err != nil {
		return nil, err
	}
	return parsed.urls, nil
}

func (p *Parser) parseSitemapContent(data []byte) (parsedSitemap, error) {
	// Try to parse as sitemap index first
	var sitemap Sitemap
	if err := xml.Unmarshal(data, &sitemap); err == nil && len(sitemap.URLs) > 0 {
		urls := make([]string, len(sitemap.URLs))
		for i, url := range sitemap.URLs {
			urls[i] = url.Loc
		}
		return parsedSitemap{urls: urls, isIndex: true}, nil
	}

	// Try to parse as URL set
	var urlSet URLSet
	if err := xml.Unmarshal(data, &urlSet); err == nil && len(urlSet.URLs) > 0 {
		urls := make([]string, len(urlSet.URLs))
		for i, url := range urlSet.URLs {
			urls[i] = url.Loc
		}
		return parsedSitemap{urls: urls}, nil
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
		return parsedSitemap{urls: urls}, nil
	}

	return parsedSitemap{}, fmt.Errorf("unable to parse sitemap format")
}

func addUniqueURLs(target []string, urls []string, seen map[string]bool) []string {
	for _, url := range urls {
		if seen[url] {
			continue
		}
		seen[url] = true
		target = append(target, url)
	}
	return target
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
