package main

import (
	"fmt"
	"os"

	"github.com/benvon/sitemap-crawler/internal/config"
	"github.com/benvon/sitemap-crawler/internal/crawler"
	"github.com/sirupsen/logrus"
)

// Version information (set by GoReleaser)
var (
	version = "dev"
	commit  = "dev"
	date    = "dev"
	builtBy = "dev"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Set up logging
	logger := logrus.New()
	if cfg.Debug {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Log version information
	logger.WithFields(logrus.Fields{
		"version": version,
		"commit":  commit,
		"date":    date,
		"builtBy": builtBy,
	}).Info("Starting sitemap crawler")

	// Create and run crawler
	c := crawler.New(cfg, logger)
	if err := c.Run(); err != nil {
		logger.WithError(err).Fatal("Crawler failed")
	}
}
