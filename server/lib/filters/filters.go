package filters

import (
	"context"
	"fmt"
	"git.nunosempere.com/NunoSempere/news/lib/types"
	"github.com/jackc/pgx/v5"
	"github.com/adrg/strutil/metrics"
	"log"
	"net/url"
	"slices"
	"strings"
	"time"
)

// DuplicateEntry represents an entry in the duplicates table
type DuplicateEntry struct {
	ID           int
	OriginalTitle string
	CleanedTitle string
	Link         string
	FirstSeenAt  time.Time
}

// IsDupe checks if a source is a duplicate by comparing title and link with existing entries
// It uses both exact matching and similarity metrics for improved duplicate detection
func IsDupe(source types.Source, database_url string) bool {
	conn, err := pgx.Connect(context.Background(), database_url)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		return false
	}
	defer conn.Close(context.Background())

	// Check for exact duplicates first (case-insensitive)
	var existsExact bool
	err = conn.QueryRow(context.Background(), `
		SELECT EXISTS(
			SELECT 1 FROM sources 
			WHERE UPPER(title) = UPPER($1) OR link = $2
		)
	`, source.Title, source.Link).Scan(&existsExact)
	
	if err != nil {
		log.Printf("Error checking for exact duplicates: %v\n", err)
		return false
	}

	if existsExact {
		log.Printf("Skipping exact duplicate title/link: %v %v\n", source.Title, source.Link)
		return true
	}

	// If not an exact duplicate, check for similar titles using the duplicates table
	cleanedTitle := CleanTitle(source.Title)
	
	// Check if we've seen a similar title before
	similarDupes, err := getSimilarTitles(conn, cleanedTitle)
	if err != nil {
		log.Printf("Error checking for similar titles: %v\n", err)
		return false
	}

	// If similar titles found, check similarity using Hamming distance
	for _, dupe := range similarDupes {
		if isTitleSimilar(cleanedTitle, dupe.CleanedTitle) {
			log.Printf("Skipping similar title: \"%s\" matches \"%s\"\n", 
				cleanedTitle, dupe.CleanedTitle)
			
			// Store this as a duplicate for future reference
			saveAsDuplicate(conn, source.Title, cleanedTitle, source.Link)
			return true
		}
	}

	// Not a duplicate, save to duplicates table for future reference
	saveAsDuplicate(conn, source.Title, cleanedTitle, source.Link)
	log.Printf("Article is not a duplicate")
	return false
}

// getSimilarTitles retrieves potential similar titles from the duplicates table
func getSimilarTitles(conn *pgx.Conn, cleanedTitle string) ([]DuplicateEntry, error) {
	var results []DuplicateEntry
	
	// Implement a query that looks for potential matches
	// This could be based on word overlap, prefix matching, or other heuristics
	// For now, we'll use a simple LIKE query with the first few words
	
	words := strings.Fields(cleanedTitle)
	if len(words) < 3 {
		return results, nil
	}
	
	// Create a search pattern with the first 3 words
	searchPattern := fmt.Sprintf("%s %s %s%%", words[0], words[1], words[2])
	
	rows, err := conn.Query(context.Background(), `
		SELECT id, original_title, cleaned_title, link, first_seen_at 
		FROM duplicates 
		WHERE cleaned_title LIKE $1
		ORDER BY first_seen_at DESC
		LIMIT 10
	`, searchPattern)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var entry DuplicateEntry
		err := rows.Scan(&entry.ID, &entry.OriginalTitle, &entry.CleanedTitle, &entry.Link, &entry.FirstSeenAt)
		if err != nil {
			return nil, err
		}
		results = append(results, entry)
	}
	
	return results, nil
}

// isTitleSimilar checks if two titles are similar using Hamming distance
func isTitleSimilar(title1, title2 string) bool {
	// If titles are too different in length, they're probably not similar
	lenDiff := abs(len(title1) - len(title2))
	if lenDiff > 10 {
		return false
	}
	
	// Use Hamming distance for titles with similar length
	minLength := min(len(title1), len(title2))
	if minLength > 20 {
		hamming := metrics.NewHamming()
		// Compare the first 30 chars or the minimum length
		compareLength := min(30, minLength)
		distance := hamming.Distance(title1[:compareLength], title2[:compareLength])
		
		// If the distance is small enough, consider them similar
		// The threshold can be adjusted based on testing
		return distance <= 5
	}
	
	return false
}

// saveAsDuplicate saves an entry to the duplicates table
func saveAsDuplicate(conn *pgx.Conn, originalTitle, cleanedTitle, link string) {
	_, err := conn.Exec(context.Background(), `
		INSERT INTO duplicates (original_title, cleaned_title, link, first_seen_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (cleaned_title) DO NOTHING
	`, originalTitle, cleanedTitle, link, time.Now())
	
	if err != nil {
		log.Printf("Error saving to duplicates table: %v\n", err)
	}
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IsGoodHost checks if the source URL is from an allowed host
func IsGoodHost(source types.Source) bool {
	parsedURL, err := url.Parse(source.Link)
	if err != nil {
		log.Printf("Error parsing link: %v", err)
		return false
	}
	skippable_hosts := []string{"www.washingtonpost.com", "www.youtube.com", "www.naturalnews.com", "facebook.com", "m.facebook.com"}
	is_bad_host := slices.Contains(skippable_hosts, parsedURL.Host)
	if is_bad_host {
		log.Printf("Article is from a bad host")
	} else {
		log.Printf("Article is from a good host")
	}

	return !is_bad_host
}

// CleanTitle0 removes text after a specific marker if it appears after a minimum length
func CleanTitle0(s string, endingMarker string) string {
	// endingMarkers: "-", "|"
	result := s
	
	// Only crop if the title is long enough to be meaningful after cropping
	minLengthBeforeMarker := 15 // Ensure we have a meaningful title
	
	if pos := strings.Index(s, endingMarker); pos != -1 && pos >= minLengthBeforeMarker {
		result = s[:pos]
	}
	
	return result
}

// CleanTitle applies multiple cleaning operations to a title
func CleanTitle(s string) string {
	s2 := CleanTitle0(s, " – ")
	s3 := CleanTitle0(s2, " - ")
	s4 := CleanTitle0(s3, "|")
	s5 := strings.TrimSpace(s4)
	return s5
}