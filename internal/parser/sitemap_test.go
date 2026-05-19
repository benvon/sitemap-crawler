package parser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewParser(t *testing.T) {
	t.Parallel()

	timeout := 30 * time.Second
	p := NewParser(timeout)

	if p == nil {
		t.Fatal("Parser should not be nil")
	}

	if p.client.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, p.client.Timeout)
	}
}

func TestValidateURL(t *testing.T) {
	t.Parallel()

	p := NewParser(30 * time.Second)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"valid http", "http://example.com", true},
		{"valid https", "https://example.com", true},
		{"valid with path", "https://example.com/path", true},
		{"valid with query", "https://example.com?param=value", true},
		{"invalid scheme", "ftp://example.com", false},
		{"invalid format", "not-a-url", false},
		{"empty string", "", false},
		{"relative path", "/relative/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ValidateURL(tt.url)
			if result != tt.expected {
				t.Errorf("ValidateURL(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestParseXML(t *testing.T) {
	t.Parallel()

	p := NewParser(30 * time.Second)

	tests := []struct {
		name         string
		xmlData      []byte
		expectedURLs int
		expectError  bool
	}{
		{
			name: "sitemap index",
			xmlData: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>https://example.com/sitemap1.xml</loc>
		<lastmod>2023-01-01T00:00:00Z</lastmod>
	</sitemap>
	<sitemap>
		<loc>https://example.com/sitemap2.xml</loc>
		<lastmod>2023-01-02T00:00:00Z</lastmod>
	</sitemap>
</sitemapindex>`),
			expectedURLs: 2,
			expectError:  false,
		},
		{
			name: "urlset",
			xmlData: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://example.com/page1</loc>
		<lastmod>2023-01-01T00:00:00Z</lastmod>
		<changefreq>daily</changefreq>
		<priority>0.8</priority>
	</url>
	<url>
		<loc>https://example.com/page2</loc>
		<lastmod>2023-01-02T00:00:00Z</lastmod>
		<changefreq>weekly</changefreq>
		<priority>0.6</priority>
	</url>
</urlset>`),
			expectedURLs: 2,
			expectError:  false,
		},
		{
			name: "plain text",
			xmlData: []byte(`https://example.com/page1
https://example.com/page2
https://example.com/page3`),
			expectedURLs: 3,
			expectError:  false,
		},
		{
			name:         "invalid xml",
			xmlData:      []byte(`<invalid>xml</invalid>`),
			expectedURLs: 0,
			expectError:  true,
		},
		{
			name:         "empty data",
			xmlData:      []byte{},
			expectedURLs: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := p.parseXML(tt.xmlData)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(urls) != tt.expectedURLs {
				t.Errorf("Expected %d URLs, got %d", tt.expectedURLs, len(urls))
			}
		})
	}
}

func TestParseSitemapRecursiveIndex(t *testing.T) {
	t.Parallel()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "TestCrawler/1.0" {
			t.Errorf("Expected custom user agent, got %q", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("X-Test") != "yes" {
			t.Errorf("Expected custom header, got %q", r.Header.Get("X-Test"))
		}

		switch r.URL.Path {
		case "/sitemap.xml":
			if _, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap><loc>%s/one.xml</loc></sitemap>
	<sitemap><loc>%s/two.xml</loc></sitemap>
</sitemapindex>`, server.URL, server.URL); err != nil {
				t.Errorf("Failed to write sitemap index: %v", err)
			}
		case "/one.xml":
			if _, err := fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
	<url><loc>https://example.com/page2</loc></url>
</urlset>`); err != nil {
				t.Errorf("Failed to write URL set: %v", err)
			}
		case "/two.xml":
			if _, err := fmt.Fprint(w, `https://example.com/page2
https://example.com/page3`); err != nil {
				t.Errorf("Failed to write plain text sitemap: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := NewParser(30 * time.Second)
	p.SetUserAgent("TestCrawler/1.0")

	urls, err := p.ParseSitemap(server.URL+"/sitemap.xml", map[string]string{"X-Test": "yes"})
	if err != nil {
		t.Fatalf("ParseSitemap returned error: %v", err)
	}

	expected := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}
	if len(urls) != len(expected) {
		t.Fatalf("Expected %d URLs, got %d: %v", len(expected), len(urls), urls)
	}
	for i, expectedURL := range expected {
		if urls[i] != expectedURL {
			t.Errorf("Expected URL %d to be %q, got %q", i, expectedURL, urls[i])
		}
	}
}

func TestFetchAndParseRejectsOversizedSitemap(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, strings.Repeat("x", maxSitemapBytes+1)); err != nil {
			t.Errorf("Failed to write oversized sitemap: %v", err)
		}
	}))
	defer server.Close()

	p := NewParser(30 * time.Second)
	_, err := p.fetchAndParse(server.URL, nil)
	if err == nil {
		t.Fatal("Expected oversized sitemap error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("Expected maximum size error, got %v", err)
	}
}

func TestIsSitemapIndex(t *testing.T) {
	t.Parallel()

	p := NewParser(30 * time.Second)

	tests := []struct {
		name     string
		urls     []string
		expected bool
	}{
		{
			name:     "sitemap urls",
			urls:     []string{"https://example.com/sitemap1.xml", "https://example.com/sitemap2.xml"},
			expected: true,
		},
		{
			name:     "mixed urls",
			urls:     []string{"https://example.com/sitemap.xml", "https://example.com/page1"},
			expected: true,
		},
		{
			name:     "regular urls",
			urls:     []string{"https://example.com/page1", "https://example.com/page2"},
			expected: false,
		},
		{
			name:     "empty urls",
			urls:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.isSitemapIndex(tt.urls)
			if result != tt.expected {
				t.Errorf("isSitemapIndex(%v) = %v, expected %v", tt.urls, result, tt.expected)
			}
		})
	}
}

func TestURLStruct(t *testing.T) {
	t.Parallel()

	// Test URL struct marshaling
	url := URL{
		Loc:        "https://example.com",
		LastMod:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		ChangeFreq: "daily",
		Priority:   0.8,
	}

	if url.Loc != "https://example.com" {
		t.Errorf("Expected Loc %s, got %s", "https://example.com", url.Loc)
	}

	if url.ChangeFreq != "daily" {
		t.Errorf("Expected ChangeFreq %s, got %s", "daily", url.ChangeFreq)
	}

	if url.Priority != 0.8 {
		t.Errorf("Expected Priority %f, got %f", 0.8, url.Priority)
	}
}
