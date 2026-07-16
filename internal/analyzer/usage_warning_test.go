package analyzer

import (
	"testing"

	"github.com/777genius/claude-notifications/pkg/jsonl"
)

func TestDetectUsageWarning(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"approaching usage limit", "Heads up: you're approaching your usage limit for this window.", true},
		{"nearing usage limit", "You are nearing your usage limit.", true},
		{"case insensitive", "APPROACHING THE USAGE LIMIT now", true},
		{"normal completion", "All done — the branch is pushed.", false},
		{"session reached is not warning", "Session limit reached.", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msgs := []jsonl.Message{assistantMsg(c.text)}
			if got := detectUsageWarning(msgs); got != c.want {
				t.Errorf("detectUsageWarning(%q) = %v, want %v", c.text, got, c.want)
			}
		})
	}
}
