package readability

import (
	"errors"
	"git.nunosempere.com/NunoSempere/news/lib/web"
	"log"
	"net/url"
	"net/http"
	"strings"
	"github.com/PuerkitoBio/goquery"
)

func GetReadabilityOutput(article_url string) (string, error) {
	readability_url := "https://trastos.nunosempere.com/readability?url=" + article_url // url must start with https
	readability_response, err := web.Get(readability_url)
	if err != nil {
		return "", err
	}
	readability_result := string(readability_response)

	if len(readability_result) < 200 {
		log.Println("Error in GetReadabilityOutput: readability output too short")
		return "", errors.New("Readability output too short")
	}
	return readability_result, nil
}

func ReplaceWithOSFrontend(u string) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(u)
	if err != nil {
		log.Printf("Error parsing url to check for open source alternatives: %v", err)
		return "", err
	}

	// Check if the host is bad_domain.com
	if parsedURL.Host == "www.reuters.com" {
		// Replace the host with open_source_alternative.net
		parsedURL.Host = "neuters.de"

		// Return the updated URL as a string
		return parsedURL.String(), nil
	}
	// Return the original URL if no replacement is necessary
	return u, nil
}

// Try to extract title from HTML
func ExtractTitle(url string) string {
	url_for_title := url 
	oss_url, err := ReplaceWithOSFrontend(url)
	if err == nil {
		url_for_title = oss_url
	}
	resp, err := http.Get(url_for_title)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	title := doc.Find("title").Text()
	return strings.TrimSpace(title)
}

func GetArticleContent(init_url string) (string, error) {
	req_url := init_url
	os_url, err0 := ReplaceWithOSFrontend(init_url)
	if err0 == nil {
		req_url = os_url
	}
	readable_text, err1 := GetReadabilityOutput(req_url)
	log.Printf("Req url: %v", req_url)
	if err1 != nil {

		url_content, err2 := web.Get(req_url)
		if err2 != nil {
			log.Println("Errors in both redability AND web.Get")
			err := errors.Join(err1, err2)
			return "", err
		}
		compressed_html, err2 := web.CompressHtml(string(url_content[:]))
		if err2 != nil {
			log.Println("Errors in both redability AND web.Get")
			err := errors.Join(err1, err2)
			return "", err
		}
		return compressed_html, nil

	}
	return readable_text, nil
}

/*
func main() {
	url := "https://www.washingtonpost.com/nation/2024/02/29/ukraine-support-alabama-political-divide/"
	readable_content, err := getReadabilityOutput(url)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(readable_content)
	}

	url = "https://www.vox.com/future-perfect/2024/2/13/24070864/samotsvety-forecasting-superforecasters-tetlock"
	readable_content, err = getReadabilityOutput(url)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(readable_content)
	}
}
*/
