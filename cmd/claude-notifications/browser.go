package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/777genius/claude-notifications/internal/browserserve"
	"github.com/777genius/claude-notifications/internal/config"
	"github.com/777genius/claude-notifications/internal/errorhandler"
	"github.com/777genius/claude-notifications/internal/logging"
)

// runBrowserServe starts the localhost listener for the Chrome extension.
func runBrowserServe() {
	defer errorhandler.HandlePanic()

	pluginRoot := getPluginRoot()
	if _, err := logging.InitLogger(pluginRoot); err == nil {
		defer logging.Close()
	}

	token := browserserve.LoadToken()
	if token == "" {
		fmt.Fprintln(os.Stderr, "No browser-listener token found. Run: claude-notifications install-browser-listener")
		os.Exit(1)
	}

	cfg, err := config.LoadFromPluginRoot(pluginRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load failed: %v\n", err)
		os.Exit(1)
	}

	srv := browserserve.New(cfg, token)
	if err := srv.ListenAndServe(browserserve.DefaultPort); err != nil {
		fmt.Fprintf(os.Stderr, "browser listener stopped: %v\n", err)
		os.Exit(1)
	}
}

// runInstallBrowserListener generates the shared token, installs a LaunchAgent
// that keeps the listener running, and prints the token for the user to paste
// into the Chrome extension.
func runInstallBrowserListener() {
	defer errorhandler.HandlePanic()

	// 1. Generate (or reuse) the shared token.
	token := browserserve.LoadToken()
	if token == "" {
		buf := make([]byte, 24)
		if _, err := rand.Read(buf); err != nil {
			fmt.Fprintf(os.Stderr, "token generation failed: %v\n", err)
			os.Exit(1)
		}
		token = hex.EncodeToString(buf)
		tokenPath := browserserve.TokenPath()
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "cannot create token dir: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(tokenPath, []byte(token), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "cannot write token: %v\n", err)
			os.Exit(1)
		}
	}

	// 2. Install the LaunchAgent (macOS) so the listener auto-starts.
	if currentGOOS == "darwin" {
		if err := installBrowserLaunchAgent(); err != nil {
			fmt.Fprintf(os.Stderr, "LaunchAgent install failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "You can still run the listener manually: claude-notifications serve")
		}
	}

	fmt.Println("Browser listener installed.")
	fmt.Printf("\nPaste this token into the Claude Notifications Chrome extension:\n\n  %s\n\n", token)
	fmt.Printf("Listener URL: http://127.0.0.1:%d\n", browserserve.DefaultPort)
}

// installBrowserLaunchAgent writes and loads a LaunchAgent plist that runs
// `claude-notifications serve` and keeps it alive.
func installBrowserLaunchAgent() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	label := "com.claude-notifications.browser-listener"
	plistPath := filepath.Join(home, "Library", "LaunchAgents", label+".plist")
	logPath := filepath.Join(home, ".claude", "browser-listener.log")

	// The LaunchAgent runs outside the plugin's hook environment, so bake in
	// CLAUDE_PLUGIN_ROOT — GetTerminalNotifierPath relies on it to locate
	// ClaudeNotifier.app.
	pluginRoot := getPluginRoot()

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>serve</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>CLAUDE_PLUGIN_ROOT</key><string>%s</string>
    </dict>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
    <key>StandardOutPath</key><string>%s</string>
    <key>StandardErrorPath</key><string>%s</string>
</dict>
</plist>
`, label, exe, pluginRoot, logPath, logPath)

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		return err
	}

	// Reload: bootout then bootstrap (ignore bootout error when not loaded).
	uid := fmt.Sprintf("gui/%d", os.Getuid())
	_ = exec.Command("launchctl", "bootout", uid+"/"+label).Run()
	if err := exec.Command("launchctl", "bootstrap", uid, plistPath).Run(); err != nil {
		// Fall back to legacy load for older macOS.
		if err2 := exec.Command("launchctl", "load", "-w", plistPath).Run(); err2 != nil {
			return fmt.Errorf("bootstrap and load both failed: %v / %v", err, err2)
		}
	}
	return nil
}
