package analyzer

import (
	"regexp"
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

// courtesyCloserRe matches polite sign-offs that contain asking-phrases but do
// not actually request input — "let me know if you want any of it changed",
// "tell me if anything's off", "feel free to ask". Matched spans are removed
// before the phrase scan so closers never classify a turn as a question.
// A message that ENDS with "?" is still a question regardless.
var courtesyCloserRe = regexp.MustCompile(`(?i)((let me know|tell me|say the word|shout|ping me)\s+if\b[^.!?\n]*|feel free to[^.!?\n]*|if you (have|need|want)[^.!?\n]*(question|help|anything|else)[^.!?\n]*|happy to (help|assist)[^.!?\n]*)`)

// awaitingUserResponse reports whether the final assistant message asks the
// user for input: it ends with a question, or contains an explicit
// waiting-on-you phrase. Courtesy closers ("let me know if you need anything")
// are stripped first — they are sign-offs, not requests. Used to reclassify
// tool-heavy turns that finish by asking something ("done X — approve the
// push?") from task_complete to question, so the notification says
// "Input needed" instead of "Done".
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

	lower := strings.ToLower(courtesyCloserRe.ReplaceAllString(text, ""))
	for _, phrase := range awaitingUserPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}
