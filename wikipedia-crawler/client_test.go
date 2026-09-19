package jev

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDecide(t *testing.T) {
	client := NewClient("test-key", &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" || r.URL.String() != "https://openrouter.ai/api/alpha/decisions" {
			t.Fatalf("unexpected endpoint: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("missing headers")
		}
		var body struct {
			Model     string
			Questions map[string]struct {
				Type     string
				Criteria json.RawMessage
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != DefaultModel {
			t.Fatalf("model: %s", body.Model)
		}
		if string(body.Questions["urgent"].Criteria) != `{"false":"No urgency","true":"Urgent"}` {
			t.Fatal("incorrect label criteria")
		}
		if string(body.Questions["frustration"].Criteria) != `["Calm","Angry"]` {
			t.Fatal("incorrect scale criteria")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"answers":{"urgent":{"type":"noul","noul":0.95},"department":{"type":"choice","choice":"billing","probabilities":{"billing":0.88}},"frustration":{"type":"score","score":1.05}}}`))}, nil
	})})
	response, err := client.Decide(context.Background(), Request{State: "Help!", Questions: map[string]Question{
		"urgent":      {Type: Noul, Criteria: Labels{"true": "Urgent", "false": "No urgency"}},
		"frustration": {Type: Score, Criteria: Scale{"Calm", "Angry"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if response.Answers["urgent"].Noul != 0.95 || response.Answers["department"].Choice != "billing" || response.Answers["department"].Probabilities["billing"] != 0.88 || response.Answers["frustration"].Score != 1.05 {
		t.Fatalf("unexpected answers: %+v", response.Answers)
	}
}

func TestDecideErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, "private response"},
		{"malformed", 200, "not json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient("test-key", &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})})
			_, err := client.Decide(context.Background(), Request{})
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), "private response") {
				t.Fatal("error leaked response body")
			}
		})
	}
}

func TestDecideNoulEscalation(t *testing.T) {
	input := Request{
		State: "I have asked three times now. Can I please just talk to a real person?",
		Questions: map[string]Question{
			"is_human_escalation": {Type: Noul, Instructions: "Is the customer asking for a human agent?"},
			"is_repeat_contact": {
				Type:         Noul,
				Instructions: "Has the customer contacted support about this before?",
				Criteria: Labels{
					"true":  "Mentions a prior attempt, ticket, or that they have asked before",
					"false": "No sign of any previous contact",
				},
			},
		},
	}
	client := NewClient("test-key", &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		var body struct {
			State     string `json:"state"`
			Questions map[string]struct {
				Type         QuestionType    `json:"type"`
				Instructions string          `json:"instructions"`
				Criteria     json.RawMessage `json:"criteria"`
			} `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.State != input.State {
			t.Fatalf("state: %q", body.State)
		}
		for name, question := range input.Questions {
			got := body.Questions[name]
			if got.Type != Noul || got.Instructions != question.Instructions {
				t.Fatalf("unexpected question %s: %+v", name, got)
			}
		}
		if len(body.Questions["is_human_escalation"].Criteria) != 0 {
			t.Fatal("absent criteria must be omitted, not sent as null")
		}
		var labels Labels
		if err := json.Unmarshal(body.Questions["is_repeat_contact"].Criteria, &labels); err != nil {
			t.Fatal(err)
		}
		for key, want := range input.Questions["is_repeat_contact"].Criteria.(Labels) {
			if labels[key] != want {
				t.Fatalf("criteria %s: %q", key, labels[key])
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"model":"jev-latest","answers":{"is_human_escalation":{"type":"noul","noul":0.99},"is_repeat_contact":{"type":"noul","noul":0.93}},"usage":{"input_tokens":360,"output_tokens":39}}`))}, nil
	})})
	response, err := client.Decide(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if response.Model != "jev-latest" || response.Usage.InputTokens != 360 || response.Usage.OutputTokens != 39 {
		t.Fatalf("unexpected response metadata: %+v", response)
	}

	for name, want := range map[string]float64{"is_human_escalation": 0.99, "is_repeat_contact": 0.93} {
		if got := response.Answers[name]; got.Type != Noul || got.Noul != want {
			t.Fatalf("answer %s: %+v", name, got)
		}
	}
}
