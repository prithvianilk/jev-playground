package jev

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const WikipediaBaseURL = "https://en.wikipedia.org"

// WikipediaClient fetches article pages from English Wikipedia.
type WikipediaClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewWikipediaClient creates a client. A nil HTTP client gets a sensible timeout.
func NewWikipediaClient(httpClient *http.Client) *WikipediaClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &WikipediaClient{httpClient: httpClient, baseURL: WikipediaBaseURL}
}

// ArticleURL returns the canonical URL for an article title, such as "Go (programming language)".
func (c *WikipediaClient) ArticleURL(article string) (string, error) {
	article = strings.TrimSpace(article)
	if article == "" {
		return "", fmt.Errorf("article title is required")
	}
	return strings.TrimRight(c.baseURL, "/") + "/wiki/" + url.PathEscape(article), nil
}

// GetArticle downloads the article HTML. The caller controls cancellation through ctx.
func (c *WikipediaClient) GetArticle(ctx context.Context, article string) ([]byte, error) {
	articleURL, err := c.ArticleURL(article)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, articleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Wikipedia request: %w", err)
	}
	req.Header.Set("User-Agent", "jev-playground/1.0 (Wikipedia client)")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Wikipedia article: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Wikipedia article request: HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read Wikipedia article: %w", err)
	}
	return body, nil
}
