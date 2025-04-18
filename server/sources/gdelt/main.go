package main

import (
	"github.com/joho/godotenv"
	"log"
	"os"
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
	logPath := "sources/gdelt/v2.log"
	maxSize, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_SIZE", "10"))
	maxBackups, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_BACKUPS", "5"))
	maxAge, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_AGE", "30"))
	minLevel := logger.GetLogLevelFromString(getEnvWithDefault("LOG_LEVEL", "INFO"))

	var err error
	logger, err = logger.NewLogger("GDELT", logPath, maxSize, maxBackups, maxAge, minLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	
	// Get keys
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

	// Search gkg
	ticker_gkg := time.NewTicker(15 * time.Minute)
	defer ticker_gkg.Stop()
	for ; true; <-ticker_gkg.C {
		go func() {
			// The prospector can be processing more than 2 GKG 15 minute intervals at the same time!
			logger.Info("Processing new gkg batch (this may take a min or two, as it's a large zip file)")
			articles, err := SearchGKG()
			if err != nil {
				for i := 0; i < 2; i++ {
					logger.Error("GDELT.GKG error: %v", err)
					if i != 9 {
						logger.Info("trying again in 30s")
					}
					time.Sleep(30 * time.Second)
					articles, err = SearchGKG()
					if err == nil {
						break
					}
				}
				if err != nil {
					logger.Error("GDELT.GKG error: %v", err)
					logger.Error("Tried 10 times and couldn't parse GKG zip file")
					return
				}
			}
			logger.Info("Batch has %d articles", len(articles))
			for i, article := range articles {
				logger.Info("Article #%v/%v [GDELT.GKG]: %v (%v)", i+1, len(articles), article.Title, article.Date)

				expanded_source, passes_filters := FilterAndExpandSource(article, openai_key, pg_database_url)
				if passes_filters {
					SaveSource(expanded_source)
				}
			}
			logger.Info("Finished processing gkg batch")
			return
		}()
	}
}