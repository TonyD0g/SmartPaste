package suggest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"smartpaste/internal/model"
	"smartpaste/internal/store"
)

const (
	NoMatchCandidate  = "__no_good_match__"
	defaultEndpoint   = "https://api.typesafe.ai/v1/systemone"
	MinimumConfidence = 0.4
)

type Client struct {
	HTTP     *http.Client
	Endpoint string
}

func New() *Client {
	endpoint := os.Getenv("SMARTPASTE_TYPESAFE_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Client{
		HTTP:     &http.Client{Timeout: 12 * time.Second},
		Endpoint: endpoint,
	}
}

func (c *Client) Suggest(ctx context.Context, apiKey string, field model.Field, candidates []model.Candidate) model.Outcome {
	if strings.TrimSpace(apiKey) == "" {
		return model.Outcome{Failed: true}
	}

	criteria := make(map[string]string, len(candidates)+1)
	for _, candidate := range candidates {
		criteria[candidate.Text] = candidate.ID
	}
	criteria[NoMatchCandidate] = "None of the saved clipboard values is an appropriate answer for this field."

	question := map[string]any{
		"model":       "jev-latest",
		"state":       "focused_field",
		"label":       field.Label,
		"role":        field.Role,
		"application": field.App,
		"page_url":    nil,
		"candidates":  candidates,
		"policy": "Suggest only a value that naturally and specifically answers the " +
			"focused field. Prefer no match over a merely plausible or unrelated value. " +
			"Never transform, combine, or invent text.",
	}

	body := map[string]any{
		"suggestion": map[string]any{
			"type": "choice",
			"instructions": "Which one saved clipboard value is the best exact value to " +
				"enter in `focused_field`? Choose its candidate id. Choose the no-match " +
				"option when none clearly belongs in this field.",
			"criteria": criteria,
		},
		"questions": []any{question},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return model.Outcome{Failed: true}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return model.Outcome{Failed: true}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))

	response, err := c.HTTP.Do(request)
	if err != nil {
		return model.Outcome{Failed: true}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return model.Outcome{Failed: true}
	}

	var decoded struct {
		Answers *struct {
			Answer *struct {
				Choice        *string            `json:"choice"`
				Confidence    *float64           `json:"confidence"`
				Probabilities map[string]float64 `json:"probabilities"`
			} `json:"answer"`
		} `json:"answers"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&decoded); err != nil {
		return model.Outcome{Failed: true}
	}
	if decoded.Answers == nil || decoded.Answers.Answer == nil {
		return model.Outcome{NoMatch: true}
	}

	answer := decoded.Answers.Answer
	if answer.Choice == nil || *answer.Choice == "" || *answer.Choice == NoMatchCandidate {
		return model.Outcome{NoMatch: true}
	}

	choice := *answer.Choice
	var selected *model.Candidate
	for i := range candidates {
		if candidates[i].ID == choice {
			selected = &candidates[i]
			break
		}
	}
	if selected == nil {
		return model.Outcome{NoMatch: true}
	}

	probabilities := answer.Probabilities
	rankedIDs := make([]string, 0, len(probabilities))
	for id := range probabilities {
		rankedIDs = append(rankedIDs, id)
	}
	sort.Slice(rankedIDs, func(i, j int) bool {
		return probabilities[rankedIDs[i]] > probabilities[rankedIDs[j]]
	})

	confidence := 0.0
	if value, ok := probabilities[choice]; ok {
		confidence = value
	} else if answer.Confidence != nil {
		confidence = *answer.Confidence
	}

	textByID := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		textByID[candidate.ID] = candidate.Text
	}
	if len(rankedIDs) == 0 {
		rankedIDs = []string{choice}
	}
	rankedTexts := make([]string, 0, len(rankedIDs)+1)
	seen := make(map[string]struct{}, len(rankedIDs)+1)
	for _, id := range rankedIDs {
		text, ok := textByID[id]
		if !ok {
			continue
		}
		if _, duplicate := seen[strings.ToLower(text)]; duplicate {
			continue
		}
		seen[strings.ToLower(text)] = struct{}{}
		rankedTexts = append(rankedTexts, text)
	}
	if _, exists := seen[strings.ToLower(selected.Text)]; !exists {
		rankedTexts = append([]string{selected.Text}, rankedTexts...)
	}

	return model.Outcome{
		Clip: &model.Clip{
			ID:        selected.ID,
			Text:      selected.Text,
			SourceApp: selected.SourceApp,
			CreatedAt: model.NewTimestamp(time.Now()),
		},
		Confidence:  confidence,
		RankedTexts: rankedTexts,
	}
}

func CandidateValues(clips []model.Clip) []model.Candidate {
	result := make([]model.Candidate, 0, 64)
	seen := make(map[string]struct{})
	totalLength := 0

	for _, clip := range clips {
		if clip.IsImage() {
			continue
		}
		values := clip.IndexedValues
		if len(values) == 0 {
			values = store.IndexValues(clip.Text)
		}
		for index, value := range values {
			key := strings.ToLower(value)
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			if len(result) >= 180 || totalLength+len([]rune(value)) > 24_000 {
				return result
			}
			seen[key] = struct{}{}
			totalLength += len([]rune(value))
			result = append(result, model.Candidate{
				ID:        fmt.Sprintf("%s:index:%s", clip.ID, strconv.Itoa(index)),
				Text:      value,
				SourceApp: clip.SourceApp,
				CopiedAt:  clip.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
	}
	return result
}
