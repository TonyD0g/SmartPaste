package suggest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smartpaste/internal/model"
)

func TestSuggestReturnsBestCandidate(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected authorization header: %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"answers":{"answer":{"choice":"first:index:0","confidence":0.8,"probabilities":{"first:index:0":0.8,"second:index:0":0.2}}}}`))
	}))
	defer server.Close()

	client := New()
	client.Endpoint = server.URL

	candidates := []model.Candidate{
		{ID: "first:index:0", Text: "alice@example.com", SourceApp: "Mail", CopiedAt: time.Now().Format(time.RFC3339)},
		{ID: "second:index:0", Text: "bob@example.com", SourceApp: "Mail", CopiedAt: time.Now().Format(time.RFC3339)},
	}
	outcome := client.Suggest(context.Background(), "test-key", model.Field{
		Label: "Email",
		Role:  "Edit",
		App:   "Mail",
	}, candidates)

	if outcome.Failed || outcome.NoMatch || outcome.Clip == nil {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
	if outcome.Clip.Text != "alice@example.com" {
		t.Fatalf("unexpected clip text: %q", outcome.Clip.Text)
	}
	if outcome.Confidence != 0.8 {
		t.Fatalf("unexpected confidence: %v", outcome.Confidence)
	}
	if len(outcome.RankedTexts) != 2 || outcome.RankedTexts[0] != "alice@example.com" {
		t.Fatalf("unexpected ranking: %v", outcome.RankedTexts)
	}

	suggestion, ok := requestBody["suggestion"].(map[string]any)
	if !ok {
		t.Fatalf("missing suggestion in request")
	}
	criteria, _ := suggestion["criteria"].(map[string]any)
	if criteria["alice@example.com"] != "first:index:0" {
		t.Fatalf("criteria did not map values to ids: %v", criteria)
	}
	if criteria[NoMatchCandidate] == nil {
		t.Fatalf("criteria missing no-match option")
	}
}

func TestSuggestNoMatchAndFailure(t *testing.T) {
	noMatch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"answers":{"answer":{"choice":"__no_good_match__","confidence":0.9}}}`))
	}))
	defer noMatch.Close()

	client := New()
	client.Endpoint = noMatch.URL
	outcome := client.Suggest(context.Background(), "key", model.Field{}, []model.Candidate{{ID: "a", Text: "value"}})
	if !outcome.NoMatch {
		t.Fatalf("expected no match, got %+v", outcome)
	}

	failure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failure.Close()
	client.Endpoint = failure.URL
	outcome = client.Suggest(context.Background(), "key", model.Field{}, []model.Candidate{{ID: "a", Text: "value"}})
	if !outcome.Failed {
		t.Fatalf("expected failure, got %+v", outcome)
	}
}

func TestCandidateValues(t *testing.T) {
	clip := model.Clip{
		ID:            "clip-1",
		Text:          "https://example.com and hello@example.com",
		SourceApp:     "Safari",
		CreatedAt:     model.NewTimestamp(time.Now()),
		IndexedValues: []string{"https://example.com", "hello@example.com"},
	}
	duplicate := model.Clip{
		ID:            "clip-2",
		Text:          "https://EXAMPLE.com",
		SourceApp:     "Safari",
		CreatedAt:     model.NewTimestamp(time.Now()),
		IndexedValues: []string{"https://EXAMPLE.com"},
	}
	candidates := CandidateValues([]model.Clip{clip, duplicate})
	if len(candidates) != 2 {
		t.Fatalf("expected deduplicated candidates, got %v", candidates)
	}
	if candidates[0].ID != "clip-1:index:0" || candidates[0].Text != "https://example.com" {
		t.Fatalf("unexpected first candidate: %+v", candidates[0])
	}
	if candidates[0].CopiedAt == "" || candidates[0].SourceApp != "Safari" {
		t.Fatalf("missing metadata: %+v", candidates[0])
	}
}
