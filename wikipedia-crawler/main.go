package jev

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
)

// Main crawls Wikipedia article titles matched by articlePattern and asks Jev
// whether each match is a pet. Keep articlePattern empty until you add the
// link pattern.
func Main() error {
	args := os.Args[1:]
	if len(args) < 2 {
		return fmt.Errorf("usage: Main <initial-article> <max-level>")
	}
	maxLevel, err := strconv.Atoi(args[1])
	if err != nil || maxLevel < 0 {
		return fmt.Errorf("max-level must be a non-negative integer")
	}

	articlePattern := "https://en.wikipedia.org/wiki/([a-zA-Z0-9._%+-]+)"
	re, err := regexp.Compile(articlePattern)
	if err != nil {
		return fmt.Errorf("compile article regex: %w", err)
	}
	// An empty regex matches every position, so wait until it is supplied.
	if articlePattern == "" {
		return nil
	}

	ctx := context.Background()
	jevClient := NewClient(os.Getenv("OPENROUTER_API_KEY"), &http.Client{})
	wikiClient := NewWikipediaClient(nil)
	articles := []string{args[0]}
	visited := make(map[string]bool)

	for level := 0; len(articles) > 0 && level < maxLevel; level++ {
		fmt.Printf("level: %d\n", level)
		nextArticles := make([]string, 0)
		candidates := make([]string, 0)
		for _, article := range articles {
			if visited[article] {
				continue
			}
			visited[article] = true
			pageHTML, err := wikiClient.GetArticle(ctx, article)
			if err != nil {
				return err
			}
			for _, match := range re.FindAllStringSubmatch(string(pageHTML), -1) {
				if len(match) == 0 || match[0] == "" {
					continue
				}
				candidate := match[0]
				if len(match) > 1 {
					candidate = match[1]
				}
				candidates = append(candidates, candidate)
			}
		}

		accepted := make(chan string, len(candidates))
		errors := make(chan error, len(candidates))
		var workers sync.WaitGroup
		for _, candidate := range candidates {
			candidate := candidate
			workers.Add(1)
			go func() {
				defer workers.Done()
				response, err := jevClient.Decide(ctx, Request{State: candidate, Questions: map[string]Question{
					"is_pet": {Type: Noul, Instructions: "Is this text about a pet?"},
				}})
				if err != nil {
					errors <- err
					return
				}
				if response.Answers["is_pet"].Noul > 0.5 {
					accepted <- candidate
				}
			}()
		}
		workers.Wait()
		close(accepted)
		close(errors)
		for err := range errors {
			return err
		}
		for candidate := range accepted {
			fmt.Println(candidate)
			nextArticles = append(nextArticles, candidate)
		}
		articles = nextArticles
	}
	return nil
}
