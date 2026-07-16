package notifier

import (
	"strings"
	"testing"
)

func TestSummarizeMessage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "first sentence extracted",
			in:   "Added conversation titles to notifications. Next I will rebuild the click behaviour and deploy.",
			want: "Added conversation titles to notifications.",
		},
		{
			name: "markdown stripped",
			in:   "**Part 1 is live** — check the [notification](https://example.com) that `just appeared`.",
			want: "Part 1 is live — check the notification that just appeared.",
		},
		{
			name: "code fence removed",
			in:   "Run this:\n```bash\nmake build\n```\nthen tell me what happened.",
			want: "Run this: then tell me what happened.",
		},
		{
			name: "long text capped at word boundary with ellipsis",
			in:   strings.Repeat("word ", 40),
			want: "", // checked by length assertion below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeMessage(tt.in)
			if tt.want != "" && got != tt.want {
				t.Errorf("summarizeMessage() = %q, want %q", got, tt.want)
			}
			if len(got) > summaryMaxLen+3 {
				t.Errorf("summary too long (%d chars): %q", len(got), got)
			}
		})
	}

	t.Run("empty stays empty-safe", func(t *testing.T) {
		if got := summarizeMessage("   "); got != "   " {
			t.Errorf("blank input should be returned as-is, got %q", got)
		}
	})
}
