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
	logPath := "sources/galerts/v2.log"
	maxSize, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_SIZE", "10"))
	maxBackups, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_BACKUPS", "5"))
	maxAge, _ := strconv.Atoi(getEnvWithDefault("LOG_MAX_AGE", "30"))
	minLevel := logger.GetLogLevelFromString(getEnvWithDefault("LOG_LEVEL", "INFO"))

	var err error
	logger, err = logger.NewLogger("GALERTS", logPath, maxSize, maxBackups, maxAge, minLevel)
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

	keywords := []string{"War", "Emergency", "disaster", "alert", "nuclear", "combat duty", "human-to-human", "pandemic", "blockade", "invasion", "undersea cables", "nuclear", "Carrington event", "mystery pneumonia", "Taiwan", "Ukraine", "OpenAI announces AGI", "AI rights", "military exercise", "Kessler syndrome", "Cyberattack"}
	for true {
		logger.Info("(Re)starting Google Alerts keyword loop")
		for _, keyword := range keywords {
			logger.Info("Keyword: %v", keyword)
			articles, err := SearchGoogleAlerts(keyword)
			if err != nil {
				logger.Error("Google Alerts error: %v", err)
				continue
			}

			logger.Info("Number of articles in keyword: %v", len(articles))

			for i, article := range articles {
				logger.Info("Article #%v/%v [keyword \"%v\"]: %v (%v)", i, len(articles), keyword, article.Title, article.Date)
				expanded_source, passes_filters := FilterAndExpandSource(article, openai_key, pg_database_url)
				if passes_filters {
					SaveSource(expanded_source)
				}
			}
		}
		logger.Info("Finished Google Alerts batch, pausing for half an hour")
		time.Sleep(1800 * time.Second) // stagger a little bit)
	}
}