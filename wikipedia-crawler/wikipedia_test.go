package jev

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWikipediaClientGetArticle(t *testing.T) {
	client := NewWikipediaClient(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/wiki/Go%20%28programming%20language%29" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("missing user agent")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("<html>wikipedia</html>"))}, nil
	})})
	body, err := client.GetArticle(context.Background(), "Go (programming language)")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "<html>wikipedia</html>" {
		t.Fatalf("body: %q", body)
	}
}

func TestWikipediaClientRequiresArticle(t *testing.T) {
	client := NewWikipediaClient(nil)
	if _, err := client.GetArticle(context.Background(), " "); err == nil {
		t.Fatal("expected empty title error")
	}
}
