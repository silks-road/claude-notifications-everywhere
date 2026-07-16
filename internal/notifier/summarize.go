package notifier

import (
	"regexp"
	"strings"
)

var (
	codeFenceRe    = regexp.MustCompile("(?s)```.*?```")
	inlineCodeRe   = regexp.MustCompile("`([^`]*)`")
	mdLinkRe       = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdEmphasisRe   = regexp.MustCompile(`(\*\*|__|\*|_|~~)`)
	mdHeadingRe    = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	mdListMarkerRe = regexp.MustCompile(`(?m)^\s*([-*+]|\d+\.)\s+`)
	mdQuoteRe      = regexp.MustCompile(`(?m)^\s*>\s?`)
	whitespaceRe   = regexp.MustCompile(`\s+`)
	sentenceEndRe  = regexp.MustCompile(`[.!?](\s|$)`)
)

const summaryMaxLen = 110

// summarizeMessage reduces an assistant message to a single clean sentence
// suitable for the small notification body: markdown is stripped, the first
// sentence extracted, and the result capped at a word boundary.
func summarizeMessage(message string) string {
	s := message
	s = codeFenceRe.ReplaceAllString(s, " ")
	s = mdLinkRe.ReplaceAllString(s, "$1")
	s = inlineCodeRe.ReplaceAllString(s, "$1")
	s = mdHeadingRe.ReplaceAllString(s, "")
	s = mdListMarkerRe.ReplaceAllString(s, "")
	s = mdQuoteRe.ReplaceAllString(s, "")
	s = mdEmphasisRe.ReplaceAllString(s, "")
	s = whitespaceRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return message
	}

	// First sentence, if a boundary exists reasonably early.
	if loc := sentenceEndRe.FindStringIndex(s); loc != nil && loc[0] > 0 {
		s = strings.TrimSpace(s[:loc[0]+1])
	}

	if len(s) > summaryMaxLen {
		cut := s[:summaryMaxLen]
		if i := strings.LastIndex(cut, " "); i > summaryMaxLen/2 {
			cut = cut[:i]
		}
		s = strings.TrimRight(cut, " ,;:") + "…"
	}
	return s
}
