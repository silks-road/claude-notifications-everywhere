package analyzer

import (
	"testing"

	"github.com/777genius/claude-notifications/pkg/jsonl"
)

func assistantMsg(text string) jsonl.Message {
	return jsonl.Message{
		Type: "assistant",
		Message: jsonl.MessageContent{
			Role: "assistant",
			Content: []jsonl.Content{
				{Type: "text", Text: text},
			},
		},
	}
}

func TestAwaitingUserResponse(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"plain completion", "All tests pass and the branch is pushed.", false},
		{"ends with question mark", "Everything is committed. Shall we proceed with the deploy?", true},
		{"question mark before markdown tail", "Ready to push. Yes?**", true},
		{"approval phrase without question mark", "Everything's done locally — one approval needed from you to finish the push.", true},
		{"say the word", "The branch is ready. Say the word and I'll merge it.", true},
		{"let me know", "Deployed to staging. Let me know how it looks.", true},
		{"informational question inside but ends declarative", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msgs []jsonl.Message
			if tt.text != "" {
				msgs = []jsonl.Message{assistantMsg(tt.text)}
			}
			if got := awaitingUserResponse(msgs); got != tt.want {
				t.Errorf("awaitingUserResponse(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestAwaitingUserResponse_CourtesyClosers(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		// Merve's real false positive, 2026-07-21 (Test session)
		{"let me know if closer", "Set the dock to auto-hide and pinned the folder. Let me know if you want any of it changed.", false},
		{"tell me if closer", "All done. Tell me if anything looks off.", false},
		{"feel free closer", "Deployed. Feel free to poke around.", false},
		{"questions closer", "That completes the setup. If you have any questions, I'm here.", false},
		{"pure pleasantry", "You're welcome!", false},
		{"genuine let-me-know request", "Deployed to staging. Let me know how it looks.", true},
		{"closer plus real question still asks", "Let me know if you need anything else. Should I also enable backups?", true},
		{"question mark always wins", "Anything else you want changed?", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msgs := []jsonl.Message{assistantMsg(c.text)}
			if got := awaitingUserResponse(msgs); got != c.want {
				t.Errorf("awaitingUserResponse(%q) = %v, want %v", c.text, got, c.want)
			}
		})
	}
}
