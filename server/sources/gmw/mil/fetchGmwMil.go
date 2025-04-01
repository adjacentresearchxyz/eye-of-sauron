package main

import (
	"bytes"
	"log"
	"strings"

	"git.nunosempere.com/NunoSempere/news/lib/web"
)

func ExtractFrontpageArticle(url string) (GmwMilSource, error) {
	content, err := web.Get(url)
	if err != nil {
		return GmwMilSource{}, err
	}
	title, err := web.GetTitle(bytes.NewReader(content))
	if err != nil {
		return GmwMilSource{}, err
	}
	title = strings.TrimSpace(title)
	title, _, _ = strings.Cut(title, "\n")
	content_stripped, err := web.GetDiv(bytes.NewReader(content))
	if err != nil {
		return GmwMilSource{}, err
	}

	// Extract date from URL
	return GmwMilSource{Link: url, Content: content_stripped, Title: title}, nil
}

func GetFrontpageUrls() ([]string, error) {

	frontpageContent, err := web.Get("https://mil.gmw.cn/")
	if err != nil {
		return []string{}, err
	}
	frontpagePostUrls, err := web.GetUrls(bytes.NewReader(frontpageContent))
	if err != nil {
		return []string{}, err
	}
	var frontpageArticles []string
	for _, url := range frontpagePostUrls {
		if strings.Contains(url, "content") {
			full_article_url := url
			if !strings.Contains(url, "http") {
				full_article_url = "https://mil.gmw.cn/" + full_article_url
			}
			frontpageArticles = append(frontpageArticles, full_article_url)
		}
	}
	log.Printf("Number of articles: #%d", len(frontpageArticles))

	return frontpageArticles, nil

}
