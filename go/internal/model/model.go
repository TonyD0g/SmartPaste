package model

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// AppleReferenceDate matches Foundation's reference date. The original Swift
// app uses JSONEncoder's default date strategy, i.e. seconds since 2001-01-01.
var AppleReferenceDate = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// Clip mirrors the persisted clip model of the Swift app.
type Clip struct {
	ID               string    `json:"id"`
	Text             string    `json:"text"`
	SourceApp        string    `json:"sourceApp"`
	CreatedAt        Timestamp `json:"createdAt"`
	ImageFileName    string    `json:"imageFileName,omitempty"`
	ImagePixelWidth  int       `json:"imagePixelWidth,omitempty"`
	ImagePixelHeight int       `json:"imagePixelHeight,omitempty"`
	IndexedValues    []string  `json:"indexedValues,omitempty"`
}

func (c Clip) IsImage() bool { return c.ImageFileName != "" }

// Timestamp keeps the on-disk format compatible with the Swift application
// while still accepting RFC3339 timestamps from hand-edited files.
type Timestamp struct {
	time.Time
}

func NewTimestamp(t time.Time) Timestamp { return Timestamp{Time: t} }

func (t Timestamp) MarshalJSON() ([]byte, error) {
	seconds := t.Sub(AppleReferenceDate).Seconds()
	return []byte(strconv.FormatFloat(seconds, 'f', -1, 64)), nil
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		t.Time = time.Time{}
		return nil
	}
	if strings.HasPrefix(raw, "\"") {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		parsed, err := time.Parse(time.RFC3339Nano, text)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return err
	}
	t.Time = AppleReferenceDate.Add(time.Duration(seconds * float64(time.Second)))
	return nil
}

// Field is the subset of the focused control needed by the suggestion API.
type Field struct {
	Label string
	Role  string
	App   string
	Value string
}

// Candidate is a single value offered to Direct Jev.
type Candidate struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	SourceApp string `json:"sourceApp"`
	CopiedAt  string `json:"copiedAt"`
}

// Outcome is the result of one suggestion request.
type Outcome struct {
	Clip        *Clip
	Confidence  float64
	RankedTexts []string
	NoMatch     bool
	Failed      bool
}
