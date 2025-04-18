package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"git.nunosempere.com/NunoSempere/news/lib/types"
	"eye-of-sauron/server/lib/logger"
)

var (
	logger *logger.Logger
)

// Helper function to get environment variable with default value
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	// Initialize logging
	logPath := "sources/wikinews/v2.log"
	maxSize, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_SIZE", "10"))
	maxBackups, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_BACKUPS", "5"))
	maxAge, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_AGE", "30"))
	minLevel := logger.GetLogLevelFromString(getEnvWithDefault("LOG_LEVEL", "INFO"))

	var err error
	logger, err = logger.NewLogger("WIKINEWS", logPath, maxSize, maxBackups, maxAge, minLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Load environment variables
	err = godotenv.Load("../../.env")  // Look for .env in root directory
	if err != nil {
		err = godotenv.Load("../.env")  // Try one level up
		if err != nil {
			err = godotenv.Load(".env")  // Try current directory as fallback
			if err != nil {
				logger.Error("Error loading .env file")
				os.Exit(1)
			}
		}
	}
	openai_key := os.Getenv("OPENAI_KEY")
	pg_database_url := os.Getenv("DATABASE_POOL_URL")

	for {
		logger.Info("Starting Wikipedia current events processing")
		rssURL := "https://www.to-rss.xyz/wikipedia/current_events/"
		
		link, err := ExtractCurrentEventsLink(rssURL)
		if err != nil {
			logger.Error("Error extracting current events link: %v", err)
			continue
		}
		if link == "" {
			logger.Warning("No current events link found")
			continue
		}
		logger.Info("Current events link: %s", link)
		
		// Fetch the content
		content, err := FetchCurrentEvents(link)
		if err != nil {
			logger.Error("Error fetching current events: %v", err)
			continue
		}
		
		// Extract and process external links
		externalLinks := ExtractExternalLinks(content)
		logger.Info("Found %d external news source links", len(externalLinks))
		
		// Process each external link
		for i, extLink := range externalLinks {
			logger.Info("Processing link %d/%d: %s", i+1, len(externalLinks), extLink)
			source := types.Source{
				Title: extLink,
				Link:  extLink,
				Date:  "", // FilterAndExpandSource will set date
			}
			expanded_source, passes_filters := FilterAndExpandSource(source, openai_key, pg_database_url)
			if passes_filters {
				SaveSource(expanded_source)
			}
		}

		logger.Info("Finished processing current events, sleeping for 24 hours")
		time.Sleep(12 * time.Hour)
	}
}