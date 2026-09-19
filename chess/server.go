package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	jev "jev-playground"
)

type botMoveRequest struct {
	State string            `json:"state"`
	Moves map[string]string `json:"moves"`
}

type botMoveResponse struct {
	Move string `json:"move"`
}

type rankedChoice struct {
	Choice      string  `json:"choice"`
	Probability float64 `json:"probability"`
}

func rankChoices(probabilities map[string]float64) []rankedChoice {
	ranked := make([]rankedChoice, 0, len(probabilities))
	for choice, probability := range probabilities {
		ranked = append(ranked, rankedChoice{Choice: choice, Probability: probability})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Probability == ranked[j].Probability {
			return ranked[i].Choice < ranked[j].Choice
		}
		return ranked[i].Probability > ranked[j].Probability
	})
	return ranked
}

func main() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}

	client := jev.NewClient(apiKey, nil)
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/api/bot-move", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input botMoveRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(input.State) == "" || len(input.Moves) == 0 {
			http.Error(w, "state and legal moves are required", http.StatusBadRequest)
			return
		}
		criteria := make(jev.Labels, len(input.Moves))
		for move, description := range input.Moves {
			criteria[move] = description
		}
		result, err := client.Decide(r.Context(), jev.Request{
			Model: "jev-latest",
			State: input.State,
			Questions: map[string]jev.Question{
				"move": {
					Type:         jev.Choice,
					Instructions: "Choose the strongest legal chess move for the bot. Return exactly one move label.",
					Criteria:     criteria,
				},
			},
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("bot decision failed: %v", err), http.StatusBadGateway)
			return
		}
		answer, ok := result.Answers["move"]
		ranked, marshalErr := json.Marshal(rankChoices(answer.Probabilities))
		if marshalErr == nil {
			log.Printf("Jev choices ranked: %s", ranked)
		}
		if !ok || answer.Type != jev.Choice || answer.Choice == "" {
			http.Error(w, "Jev returned no move", http.StatusBadGateway)
			return
		}
		if _, legal := input.Moves[answer.Choice]; !legal {
			http.Error(w, "Jev selected an invalid move", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(botMoveResponse{Move: answer.Choice})
	})

	log.Println("Chess app: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
