package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const DefaultModel = "typesafe/jev-1.13"

// Client calls OpenRouter's Jev decisions endpoint. Reuse it across requests.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient uses the supplied HTTP client for timeouts and transport configuration.
func NewClient(apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{apiKey: apiKey, httpClient: httpClient}
}

type QuestionType string

const (
	Noul   QuestionType = "noul"
	Choice QuestionType = "choice"
	Score  QuestionType = "score"
)

// Criteria is either Labels for noul/choice questions or Scale for score questions.
type Criteria interface{ isCriteria() }

type Labels map[string]string

func (Labels) isCriteria() {}

type Scale []string

func (Scale) isCriteria() {}

type Question struct {
	Type         QuestionType `json:"type"`
	Instructions string       `json:"instructions"`
	Criteria     Criteria     `json:"criteria,omitempty"`
}

type Request struct {
	Model     string              `json:"model"`
	State     string              `json:"state"`
	Questions map[string]Question `json:"questions"`
}

// Answer fields are interpreted according to Type.
type Answer struct {
	Type          QuestionType       `json:"type"`
	Noul          float64            `json:"noul"`
	Choice        string             `json:"choice"`
	Score         float64            `json:"score"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type Response struct {
	Model   string            `json:"model"`
	Answers map[string]Answer `json:"answers"`
	Usage   Usage             `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Decide submits a decision request. An empty model defaults to Jev 1.13.
func (c *Client) Decide(ctx context.Context, input Request) (Response, error) {
	if input.Model == "" {
		input.Model = DefaultModel
	}
	body, err := json.Marshal(input)
	if err != nil {
		return Response{}, fmt.Errorf("encode decision request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/alpha/decisions", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("create decision request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("send decision request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Response{}, fmt.Errorf("OpenRouter decisions: HTTP %d", res.StatusCode)
	}
	var output Response
	if err := json.NewDecoder(res.Body).Decode(&output); err != nil {
		return Response{}, fmt.Errorf("decode decision response: %w", err)
	}
	return output, nil
}
