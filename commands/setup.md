---
description: Guided setup — Claude configures everything and walks you through the few macOS toggles
allowed-tools: Bash
---

# 🚀 Guided Setup

You are walking the user through setting up Claude Notifications Everywhere. Your job: run the automated setup, then guide the user through each remaining manual step **one at a time**, confirming each before moving on. Be warm, brief, and concrete.

## Step 1 — Run the automated setup

```bash
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT}"
if [ -z "$PLUGIN_ROOT" ]; then
  PLUGIN_ROOT="$HOME/.claude/plugins/marketplaces/claude-notifications-go"
fi
BIN="$PLUGIN_ROOT/bin/claude-notifications"
[ -x "$BIN" ] || BIN="$PLUGIN_ROOT/bin/claude-notifications-darwin-arm64"
"$BIN" setup
```

## Step 2 — Interpret the output and guide the user

The output marks lines as `[done]`, `[check]`, or `[you]`. For every `[you]` line, guide the user through it interactively:

1. **Notifications** (if flagged): the user opens *System Settings → Notifications → Claude Notifier*, sets **Allow** + style **Alerts**. Ask them to confirm they saw the hello notification.
2. **Browser extension** (optional — only if they use claude.ai in a browser): a Finder window with the `extension` folder opened. Walk them through: `chrome://extensions` → toggle **Developer mode** (top right) → **Load unpacked** → select that folder → click the extension's icon in the toolbar → paste the token (it's in the setup output) → **Save & test**.
3. **Accessibility**: a Finder window highlighting `ClaudeNotifier.app` and the Accessibility settings pane both opened. The user drags the app into the list and toggles it **ON**. This powers click-to-conversation and the approval buttons.
4. **Focus modes**: if they use Focus, they add **Claude** and **Claude Notifier** to its allowed apps.

Ask about ONE step, wait for confirmation, then move to the next. Skip the browser step entirely if the user says they don't use claude.ai in a browser.

## Step 3 — Verify

When all confirmed, send a test notification:

```bash
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$HOME/.claude/plugins/marketplaces/claude-notifications-go}"
BIN="$PLUGIN_ROOT/bin/claude-notifications"
[ -x "$BIN" ] || BIN="$PLUGIN_ROOT/bin/claude-notifications-darwin-arm64"
printf '%s' '{"session_id":"setup-test","cwd":"/tmp","last_assistant_message":"Setup complete - notifications are working!"}' | CLAUDE_PLUGIN_ROOT="$PLUGIN_ROOT" "$BIN" handle-hook Stop
```

If the user sees "✅ Done" with a sound: congratulate them and mention the four sounds (done / input needed / broke / approaching limit) and that clicking any notification jumps to the exact conversation.
