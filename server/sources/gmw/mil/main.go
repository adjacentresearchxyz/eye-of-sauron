package main

import (
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"time"

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
	logPath := "sources/gmw/mil/v2.log"
	maxSize, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_SIZE", "10"))
	maxBackups, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_BACKUPS", "5"))
	maxAge, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_AGE", "30"))
	minLevel := logger.GetLogLevelFromString(getEnvWithDefault("LOG_LEVEL", "INFO"))

	var err error
	logger, err = logger.NewLogger("GMW-MIL", logPath, maxSize, maxBackups, maxAge, minLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Get keys
	err = godotenv.Load("../../../../.env")  // Look for .env in root directory
	if err != nil {
		err = godotenv.Load("../../../.env")  // Try one level up
		if err != nil {
			err = godotenv.Load("../../.env")  // Try another level up
			if err != nil {
				err = godotenv.Load(".env")  // Try current directory as fallback
				if err != nil {
					logger.Error("Error loading .env file")
					os.Exit(1)
				}
			}
		}
	}
	openai_key := os.Getenv("OPENAI_KEY")
	pg_database_url := os.Getenv("DATABASE_POOL_URL")

	for {
		frontpage_articles, err := GetFrontpageUrls()
		if err != nil {
			logger.Error("%v", err)
		}
		
		titles := []string{}
		for _, url := range frontpage_articles {
			logger.Info("Url: %s", url)

			// filter here so as to not fetch full article if not necessary
			date, hasDate := ExtractDateFromURL(url)
			if hasDate && !IsWithinTwoDays(date) {
				logger.Info("Article is stale")
				continue
			}

			ms := 5000 + int64(2000 * rand.Float32())
			time.Sleep(time.Duration(ms) * time.Millisecond)
			article, err := ExtractFrontpageArticle(url)
			if err != nil {
				logger.Error("%v", err)
				continue
			}

			// All articles are duplicated, but with different underlying urls :(
			if slices.Contains(titles, article.Title) {
				logger.Debug("Article is a duplicate")
				continue
			}
			titles = append(titles, article.Title)

			logger.Info("Title: %s", article.Title)

			expanded_source, passes_filters := FilterAndExpandSource(article, openai_key, pg_database_url)
			// useful for filtering duplicates within the same batch
			// eventually I'd need to have a longer database of titles
			if passes_filters {
				logger.Info("%s", expanded_source.Summary)
				SaveSource(expanded_source)
			}
		}
		logger.Info("Finished batch. Continuing in 12 hours")
		time.Sleep(12 * time.Hour)
	}
}