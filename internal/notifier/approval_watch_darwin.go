//go:build darwin

package notifier

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/777genius/claude-notifications/internal/analyzer"
	"github.com/777genius/claude-notifications/internal/logging"
)

// The Claude desktop app handles tool permission prompts at the app level, so
// the CLI-level Notification hook (permission_prompt) never fires for Cowork
// sessions. The app does, however, log both sides of every prompt to
// ~/Library/Logs/Claude/main.log:
//
//	Emitted tool permission request <uuid> for <Tool> in session <app-session-id>
//	Received permission response for <uuid>: <decision> (tool: <Tool>)
//
// A PreToolUse hook fires just before permission evaluation, so it records the
// log offset and spawns a short-lived watcher (this file) that reads forward
// from that offset: a request for our session with no response after a grace
// period means the user is being waited on → send the 🔐 notification.

var (
	approvalEmittedRe  = regexp.MustCompile(`Emitted tool permission request ([0-9a-fA-F-]+) for (\S+) in session (\S+)`)
	approvalRespondedRe = regexp.MustCompile(`Received permission response for ([0-9a-fA-F-]+)`)
)

// desktopAppLogPath is overridable for tests.
var desktopAppLogPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Logs", "Claude", "main.log")
}

// DesktopAppLogSize returns the current size of the desktop app's main log,
// used as the starting offset for an approval watch. Returns 0 when missing.
func DesktopAppLogSize() int64 {
	info, err := os.Stat(desktopAppLogPath())
	if err != nil {
		return 0
	}
	return info.Size()
}

// approvalMarkerDir holds one marker file per notified permission request so
// concurrent watchers (one per tool call) notify at most once per request.
var approvalMarkerDir = filepath.Join(os.TempDir(), "claude-notifications-approvals")

// claimApprovalNotification atomically claims the right to notify for a
// request id. Returns false if another watcher already claimed it.
func claimApprovalNotification(requestID string) bool {
	if err := os.MkdirAll(approvalMarkerDir, 0o755); err != nil {
		return true // fail open: better a duplicate than a missed approval
	}
	f, err := os.OpenFile(filepath.Join(approvalMarkerDir, requestID), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// WatchDesktopApprovals polls the desktop app log from logOffset for
// permission requests belonging to cliSessionID and notifies for any that
// remain unanswered. Runs for up to watchWindow, polling every pollEvery.
// Intended to run in a detached process spawned from the PreToolUse hook.
func (n *Notifier) WatchDesktopApprovals(cliSessionID, cwd string, logOffset int64) error {
	appSessionID, _ := resolveDesktopSession(cliSessionID)
	if appSessionID == "" {
		return fmt.Errorf("no desktop session record for cli session %s", cliSessionID)
	}
	logPath := desktopAppLogPath()
	if logPath == "" {
		return fmt.Errorf("cannot determine desktop app log path")
	}

	const (
		watchWindow = 45 * time.Second
		pollEvery   = 2 * time.Second
		gracePeriod = 2 * time.Second
	)

	type pending struct {
		tool    string
		seenAt  time.Time
	}
	requests := map[string]*pending{} // requestID → state
	answered := map[string]bool{}

	deadline := time.Now().Add(watchWindow)
	for time.Now().Before(deadline) {
		time.Sleep(pollEvery)

		data, err := readFileFrom(logPath, logOffset)
		if err != nil {
			logging.Debug("approval-watch: log read failed: %v", err)
			continue
		}

		for _, m := range approvalRespondedRe.FindAllStringSubmatch(string(data), -1) {
			answered[m[1]] = true
		}
		for _, m := range approvalEmittedRe.FindAllStringSubmatch(string(data), -1) {
			reqID, tool, sess := m[1], m[2], m[3]
			if sess != appSessionID {
				continue
			}
			if _, known := requests[reqID]; !known {
				requests[reqID] = &pending{tool: tool, seenAt: time.Now()}
			}
		}

		for reqID, p := range requests {
			if answered[reqID] || time.Since(p.seenAt) < gracePeriod {
				continue
			}
			delete(requests, reqID)
			if !claimApprovalNotification(reqID) {
				continue
			}
			message := fmt.Sprintf("Claude wants to use %s and is waiting for your approval.", p.tool)
			logging.Debug("approval-watch: notifying for request %s (tool=%s)", reqID, p.tool)
			if err := n.SendDesktop(analyzer.StatusApprovalNeeded, message, cliSessionID, cwd); err != nil {
				logging.Warn("approval-watch: notification failed: %v", err)
			}
		}
	}
	return nil
}

// readFileFrom returns the file content from byte offset to EOF.
func readFileFrom(path string, offset int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if offset > 0 {
		if _, err := f.Seek(offset, 0); err != nil {
			return nil, err
		}
	}
	const maxRead = 4 << 20 // 4MB cap; the watch window is short
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 64*1024)
	for len(buf) < maxRead {
		n, readErr := f.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if readErr != nil {
			break
		}
	}
	return buf, nil
}
