package web

import (
	"bytes"
	"errors"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// wrap http.Get into a convenient handler
func Get(url string) ([]byte, error) {

	// log.Printf("Making request for url: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("GET error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Status error: %v", resp.StatusCode)
		return nil, errors.New("Error: http status not OK")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		return nil, err
	}
	return body, nil
}

/* Html cleanup functions */
func stripScriptsStylesAndInlineStyles(r io.Reader) (string, error) {
	var b bytes.Buffer
	z := html.NewTokenizer(r)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			// End of the document, return successfully.
			if z.Err() == io.EOF {
				return b.String(), nil
			}
			// An actual error, return with the error.
			return "", z.Err()

		case html.StartTagToken, html.SelfClosingTagToken:
			token := z.Token()
			if token.Data == "script" || token.Data == "style" {
				// Skip entire script or style block.
				findAndSkip(z, token.Data)
				continue
			}

			// Remove style attribute from the token.
			token = removeStyleAttribute(token)

			// Write the cleaned token back to buffer.
			b.WriteString(token.String())

		case html.EndTagToken:
			// Write end tag token as is.
			b.WriteString(z.Token().String())

		default:
			// For text tokens and others (like comments and doctype), write as is.
			b.WriteString(z.Token().String())
		}
	}
}

func GetUrls(body io.Reader) ([]string, error) {
	var links []string
	z := html.NewTokenizer(body)
	for {
		tt := z.Next()
		switch tt {

		case html.ErrorToken:
			if z.Err() == io.EOF {
				return links, nil
			}
			return links, z.Err()
		case html.StartTagToken, html.EndTagToken:
			token := z.Token()
			if "a" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			}
		}
	}

}

func GetTitle(body io.Reader) (string, error) {
	z := html.NewTokenizer(body)
	title := ""
	is_title := false
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return "", errors.New("No title found")
			}
			return "", z.Err()
		case html.StartTagToken:
			token := z.Token()
			if "title" == token.Data {
				is_title = true
			}
		case html.EndTagToken:
			token := z.Token()
			if "title" == token.Data {
				return title, nil
			}
		default:
			if is_title {
				title += z.Token().String()
			}
		}
	}

}

func GetDiv(body io.Reader) (string, error) {
	z := html.NewTokenizer(body)
	title := ""
	is_title := false
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return "", errors.New("No title found")
			}
			return "", z.Err()
		case html.StartTagToken:
			token := z.Token()
			if "div" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "class" && (attr.Val == "u-mainText" || attr.Val == "h-contentMain") {
						is_title = true
						break
					}
				}
			}
		case html.EndTagToken:
			token := z.Token()
			if "div" == token.Data && is_title == true {
				return title, nil
			}
		default:
			if is_title {
				title += z.Token().String()
			}
		}
	}

}

func findAndSkip(z *html.Tokenizer, tagName string) {
	// Skip all tokens until the closing tag.
	depth := 1
	for depth > 0 {
		tt := z.Next()
		switch tt {
		case html.StartTagToken:
			token := z.Token()
			if token.Data == tagName {
				depth++ // Nesting detected.
			}
		case html.EndTagToken:
			token := z.Token()
			if token.Data == tagName {
				depth-- // Closing tag found.
			}
		}
	}
}

func removeStyleAttribute(token html.Token) html.Token {
	for i := 0; i < len(token.Attr); i++ {
		if token.Attr[i].Key == "style" || token.Attr[i].Key == "class" || token.Attr[i].Key == "id" {
			// Remove element at index i.
			token.Attr = append(token.Attr[:i], token.Attr[i+1:]...)
			i-- // Adjust index as we have removed an element.
		}
	}
	return token
}

func CompressHtml(html string) (string, error) {
	reader := strings.NewReader(html)
	return stripScriptsStylesAndInlineStyles(reader)
}

func PlaintextToHTMLParagraphs(plaintext string) string {
	// Split the plaintext at each new line
	lines := strings.Split(plaintext, "\n")

	// Use a strings.Builder for efficient string concatenation
	var htmlBuilder strings.Builder

	// Iterate through each line and create a paragraph
	for _, line := range lines {
		// Skip empty lines to avoid creating empty paragraphs
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Write the paragraph tags with the line in between
		htmlBuilder.WriteString("<p>")
		htmlBuilder.WriteString(line)
		htmlBuilder.WriteString("</p>\n") // adding a newline for readability
	}

	// Return the HTML string
	return htmlBuilder.String()
}

func GetDomain(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return link
	}
	return u.Hostname()
}
