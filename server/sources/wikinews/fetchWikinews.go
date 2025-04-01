package main

import (
    "encoding/xml"
    "io"
    "net/http"
    "strings"
)

// RSS represents the root RSS structure
type RSS struct {
    XMLName xml.Name `xml:"rss"`
    Channel Channel  `xml:"channel"`
}

// Channel represents the channel element in RSS
type Channel struct {
    Items []Item `xml:"item"`
}

// Item represents each news item
type Item struct {
    Link string `xml:"link"`
}

// ExtractCurrentEventsLink gets the most recent current events link from the RSS feed URL
func ExtractCurrentEventsLink(url string) (string, error) {
    // Fetch the RSS feed
    resp, err := http.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Read and parse the XML
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    var rss RSS
    if err := xml.Unmarshal(data, &rss); err != nil {
        return "", err
    }

    // Get the first item's link (most recent)
    if len(rss.Channel.Items) > 0 {
        return rss.Channel.Items[0].Link, nil
    }

    return "", nil
}

// ExtractExternalLinks gets all external news source links from the content
func ExtractExternalLinks(content string) []string {
    var links []string
    
    // Simple string-based extraction for external links
    // Look for class="external text" href="..."
    parts := strings.Split(content, `class="external text"`)
    for _, part := range parts[1:] { // Skip first part before first match
        if idx := strings.Index(part, `href="`); idx >= 0 {
            part = part[idx+6:] // Skip 'href="'
            if idx = strings.Index(part, `"`); idx >= 0 {
                link := part[:idx]
                // Filter out Wikipedia and Wikimedia links
                if !strings.Contains(link, "wikipedia.org") && 
                   !strings.Contains(link, "wikimediafoundation.org") {
                    links = append(links, link)
                }
            }
        }
    }
    
    return links
}

// FetchCurrentEvents gets the content of the current events page
func FetchCurrentEvents(url string) (string, error) {
    resp, err := http.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    content, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    return string(content), nil
}
