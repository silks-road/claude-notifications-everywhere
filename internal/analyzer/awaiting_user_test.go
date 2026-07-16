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
