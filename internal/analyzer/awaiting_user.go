package analyzer

import (
	"strings"

	"github.com/777genius/claude-notifications/pkg/jsonl"
)

// awaitingUserPhrases are strong signals that the assistant's final message is
// asking the user for something, even when the turn used tools (which the
// state machine would otherwise classify as task_complete). Matched
// case-insensitively against the last assistant message.
var awaitingUserPhrases = []string{
	"approval needed",
	"need your approval",
	"needs your approval",
	"your go-ahead",
	"waiting on your",
	"waiting for your",
	"awaiting your",
	"let me know",
	"tell me when",
	"tell me what",
	"say the word",
	"shall i",
	"should i",
	"do you want",
	"would you like",
	"your call",
	"yes or no",
	"confirm before",
	"please confirm",
}

// awaitingUserResponse reports whether the final assistant message asks the
// user for input: it ends with a question, or contains an explicit
// waiting-on-you phrase. Used to reclassify tool-heavy turns that finish by
// asking something ("done X — approve the push?") from task_complete to
// question, so the notification says "Needs you" instead of "Done".
func awaitingUserResponse(messages []jsonl.Message) bool {
	last := jsonl.GetLastAssistantMessages(messages, 1)
	if len(last) == 0 {
		return false
	}
	text := strings.TrimSpace(jsonl.ExtractRecentText(last, 1))
	if text == "" {
		return false
	}

	// Ends with a question (ignore trailing markdown/quotes/parens).
	tail := strings.TrimRight(text, " \t\n*_`\"')")
	if strings.HasSuffix(tail, "?") {
		return true
	}

	lower := strings.ToLower(text)
	for _, phrase := range awaitingUserPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}
