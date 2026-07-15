package platform

import "testing"

func TestGetSessionSource(t *testing.T) {
	tests := []struct {
		name       string
		entrypoint string
		bundleID   string
		claudecode string
		want       SessionSource
	}{
		{"cowork desktop entrypoint", "claude-desktop", "com.anthropic.claudefordesktop", "1", SourceDesktop},
		{"cli entrypoint", "cli", "", "1", SourceCLI},
		{"sdk cli entrypoint", "sdk-cli", "", "1", SourceSDK},
		{"sdk ts entrypoint", "sdk-ts", "", "", SourceSDK},
		{"bundle id fallback without entrypoint", "", "com.anthropic.claudefordesktop", "", SourceDesktop},
		{"claudecode fallback", "", "", "1", SourceCLI},
		{"nothing set", "", "", "", SourceUnknown},
		{"terminal bundle id is not desktop", "", "com.googlecode.iterm2", "", SourceUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLAUDE_CODE_ENTRYPOINT", tt.entrypoint)
			t.Setenv("__CFBundleIdentifier", tt.bundleID)
			t.Setenv("CLAUDECODE", tt.claudecode)

			if got := GetSessionSource(); got != tt.want {
				t.Errorf("GetSessionSource() = %q, want %q", got, tt.want)
			}
		})
	}
}
