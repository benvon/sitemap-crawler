#!/bin/bash

# Test script for the sitemap crawler
# This script tests the crawler with local example files

set -e

echo "🧪 Testing Sitemap Crawler"
echo "=========================="

# Check if the binary exists
if [ ! -f "./bin/sitemap-crawler" ]; then
    echo "❌ Binary not found. Building first..."
    go build -o ./bin/sitemap-crawler ./cmd/crawler
fi

# Test 1: Basic XML sitemap parsing
echo ""
echo "📋 Test 1: Basic XML sitemap parsing"
echo "-----------------------------------"
./bin/sitemap-crawler --sitemap-url ./examples/sample-sitemap.xml --max-workers 2 --request-rate 10 --quiet

# Test 2: Sitemap index parsing
echo ""
echo "📋 Test 2: Sitemap index parsing"
echo "--------------------------------"
./bin/sitemap-crawler --sitemap-url ./examples/sitemap-index.xml --max-workers 2 --request-rate 10 --quiet

# Test 3: Plain text sitemap parsing
echo ""
echo "📋 Test 3: Plain text sitemap parsing"
echo "------------------------------------"
./bin/sitemap-crawler --sitemap-url ./examples/plain-sitemap.txt --max-workers 2 --request-rate 10 --quiet

# Test 4: JSON output format
echo ""
echo "📋 Test 4: JSON output format"
echo "-----------------------------"
./bin/sitemap-crawler --sitemap-url ./examples/sample-sitemap.xml --output-format json --max-workers 2 --request-rate 10 --quiet

# Test 5: CSV output format
echo ""
echo "📋 Test 5: CSV output format"
echo "----------------------------"
./bin/sitemap-crawler --sitemap-url ./examples/sample-sitemap.xml --output-format csv --max-workers 2 --request-rate 10 --quiet

# Test 6: Custom headers
echo ""
echo "📋 Test 6: Custom headers"
echo "-------------------------"
./bin/sitemap-crawler --sitemap-url ./examples/sample-sitemap.xml --headers "X-Test:value" --headers "User-Agent:TestBot/1.0" --max-workers 2 --request-rate 10 --quiet

echo ""
echo "✅ All tests completed successfully!"
echo ""
echo "Note: Some tests may show errors for local file URLs, which is expected."
echo "The important thing is that the tool starts and processes the sitemaps."
