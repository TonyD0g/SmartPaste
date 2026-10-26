package store

import (
	"regexp"
	"strings"
)

var (
	cardPattern       = regexp.MustCompile(`^\d{13,19}$`)
	ssnPattern        = regexp.MustCompile(`(?i)\b\d{3}-\d{2}-\d{4}\b`)
	privateKeyPattern = regexp.MustCompile(`(?i)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)
	passwordPattern   = regexp.MustCompile(`(?i)(password|passwd|passcode|密码|验证码|otp|one[- ]time code)\s*[:：=]\s*\S+`)
	indexPatterns     = []*regexp.Regexp{
		regexp.MustCompile(`(?i)https?://[^\s]+`),
		regexp.MustCompile(`(?i)[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}`),
		regexp.MustCompile(`(?i)(?:\+?\d[\d ().-]{7,}\d)`),
	}
)

// LooksSensitive reports content that must never enter history or reach the
// suggestion service: payment cards, US SSNs and private keys.
func LooksSensitive(text string) bool {
	compact := strings.NewReplacer(" ", "", "-", "").Replace(text)
	if cardPattern.MatchString(compact) {
		return true
	}
	if ssnPattern.MatchString(text) {
		return true
	}
	if privateKeyPattern.MatchString(text) {
		return true
	}
	if passwordPattern.MatchString(text) {
		return true
	}
	return strings.Contains(strings.ToLower(text), "begin private key")
}

// IndexValues extracts short, useful values from a copied text. The Swift app
// indexes URLs, e-mail addresses, phone numbers and individual lines so users
// can paste them independently later.
func IndexValues(text string) []string {
	values := make([]string, 0, 16)
	seen := make(map[string]struct{})
	totalLength := 0

	add := func(raw string) bool {
		value := strings.TrimSpace(raw)
		if value == "" || len([]rune(value)) > 500 || LooksSensitive(value) {
			return true
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return true
		}
		if len(values) >= 180 || totalLength+len([]rune(value)) > 24_000 {
			return false
		}
		seen[key] = struct{}{}
		totalLength += len([]rune(value))
		values = append(values, value)
		return true
	}

	if len([]rune(text)) < 501 {
		add(text)
	}

	for _, pattern := range indexPatterns {
		matches := pattern.FindAllString(text, 20)
		for _, match := range matches {
			if !add(match) {
				return values
			}
		}
	}

	lines := strings.Split(text, "\n")
	lineCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !add(line) {
			return values
		}
		lineCount++
		if lineCount >= 30 {
			break
		}
	}

	return values
}
