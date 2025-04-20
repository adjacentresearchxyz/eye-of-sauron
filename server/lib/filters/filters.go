package filters

import (
	"context"
	"git.nunosempere.com/NunoSempere/news/lib/types"
	"github.com/jackc/pgx/v5"
	"log"
	"net/url"
	"slices"
	"strings"
)

func IsDupe(source types.Source, database_url string) bool {
	conn, err := pgx.Connect(context.Background(), database_url)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		return false
	}
	defer conn.Close(context.Background())

	var exists bool
	err = conn.QueryRow(context.Background(), `
		SELECT EXISTS(
			SELECT 1 FROM sources 
			WHERE UPPER(title) = $1 OR link = $2
		)
	`, source.Title, source.Link).Scan(&exists)
	if err != nil {
		log.Printf("Error checking for duplicates: %v\n", err)
		return false
	}

	if exists {
		log.Printf("Skipping duplicate title/link: %v %v\n", source.Title, source.Link)
	} else {
		log.Printf("Article is not a duplicate")
	}
	return exists
}

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

func CleanTitle(s string) string {
	s2 := CleanTitle0(s, " – ")
	s3 := CleanTitle0(s2, " - ")
	s4 := CleanTitle0(s3, "|")
	s5 := strings.TrimSpace(s4)
	return s5
}

/*
func main(){
	result := "25-day-old baby pulled from Gaza rubble after Israeli strike killed family"
	result2 := CleanTitle(result, "-")
	fmt.Println(result2)
}
*/
