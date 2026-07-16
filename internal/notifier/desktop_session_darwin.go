//go:build darwin

package notifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/777genius/claude-notifications/internal/logging"
)

// desktopSessionRecord is the subset of the Claude desktop app's per-session
// metadata file we need to map a hook's session id to the app's own id.
type desktopSessionRecord struct {
	SessionID    string `json:"sessionId"`
	CLISessionID string `json:"cliSessionId"`
	Title        string `json:"title"`
	IsArchived   bool   `json:"isArchived"`
	LastActivity int64  `json:"lastActivityAt"`
}

// desktopSessionsDir returns the root the desktop app stores session metadata
// under (~/Library/Application Support/Claude/claude-code-sessions).
// Overridable for tests.
var desktopSessionsDir = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Application Support", "Claude", "claude-code-sessions")
}

// resolveDesktopSession maps the CLI-level session id received by hooks to
// the desktop app's own session record: its session id (e.g.
// "local_<other-uuid>") and the conversation title shown in the app sidebar.
//
// A conversation created in the app has sessionId != "local_"+cliSessionId;
// a session imported via the claude://resume deep link is named exactly
// "local_"+cliSessionId (a mirror). Prefer the former so callers target the
// original conversation, and fall back to the mirror only if it is all we have.
func resolveDesktopSession(cliSessionID string) (sessionID, title string) {
	root := desktopSessionsDir()
	if root == "" || cliSessionID == "" {
		return "", ""
	}

	mirrorID := "local_" + cliSessionID
	var best desktopSessionRecord
	bestActivity := int64(-1)

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		var rec desktopSessionRecord
		if json.Unmarshal(data, &rec) != nil {
			return nil
		}
		if rec.CLISessionID != cliSessionID || rec.SessionID == "" || rec.IsArchived {
			return nil
		}
		// Prefer any non-mirror record; among candidates pick the most recent.
		isMirror := rec.SessionID == mirrorID
		bestIsMirror := best.SessionID == mirrorID
		switch {
		case best.SessionID == "":
			best, bestActivity = rec, rec.LastActivity
		case bestIsMirror && !isMirror:
			best, bestActivity = rec, rec.LastActivity
		case bestIsMirror == isMirror && rec.LastActivity > bestActivity:
			best, bestActivity = rec, rec.LastActivity
		}
		return nil
	})

	if best.SessionID == "" {
		logging.Debug("No desktop session record found for cli session %s", cliSessionID)
	}
	return best.SessionID, best.Title
}

// resolveDesktopSessionID returns just the desktop app's session id for a
// CLI session id. See resolveDesktopSession.
func resolveDesktopSessionID(cliSessionID string) string {
	id, _ := resolveDesktopSession(cliSessionID)
	return id
}
