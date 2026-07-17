// background.js — service worker. Receives turn-complete events from the
// content script and forwards them to the local listener
// (claude-notifications serve) with the shared auth token.

const LISTENER_URL = "http://127.0.0.1:52741/event";

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg && msg.type === "turn_complete") {
    forward(msg.payload).then(
      (r) => sendResponse({ ok: true, result: r }),
      (e) => sendResponse({ ok: false, error: String(e) })
    );
    return true; // async response
  }
});

// Multi-tab correctness: clicking a notification runs `open <chat url>`, which
// makes Chrome open a NEW tab even when that conversation is already open
// (possibly among several claude.ai tabs / split views). When a freshly created
// tab navigates to a chat URL that another tab already shows, close the new
// tab and focus the existing one instead — the click always lands on the
// exact conversation, never a duplicate.
const newTabIds = new Set();
chrome.tabs.onCreated.addListener((tab) => newTabIds.add(tab.id));

chrome.tabs.onUpdated.addListener(async (tabId, changeInfo, tab) => {
  if (!changeInfo.url || !newTabIds.has(tabId)) return;
  const m = changeInfo.url.match(/^https:\/\/claude\.ai\/chat\/([0-9a-f-]+)/i);
  if (!m) { newTabIds.delete(tabId); return; }
  const conversationPath = "/chat/" + m[1];

  const tabs = await chrome.tabs.query({ url: "https://claude.ai/*" });
  const existing = tabs.find(
    (t) => t.id !== tabId && new URL(t.url).pathname === conversationPath
  );
  newTabIds.delete(tabId);
  if (existing) {
    await chrome.tabs.remove(tabId);
    await chrome.tabs.update(existing.id, { active: true });
    await chrome.windows.update(existing.windowId, { focused: true });
  }
});

// Tabs stop being "new" once they finish their first load.
chrome.tabs.onRemoved.addListener((tabId) => newTabIds.delete(tabId));

async function forward(payload) {
  const { token } = await chrome.storage.local.get("token");
  if (!token) {
    // Not configured yet; surface once via badge.
    chrome.action.setBadgeText({ text: "!" });
    chrome.action.setBadgeBackgroundColor({ color: "#D97757" });
    throw new Error("no token set — open the extension popup and paste it");
  }
  const resp = await fetch(LISTENER_URL, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Auth-Token": token },
    body: JSON.stringify(payload),
  });
  if (!resp.ok) throw new Error("listener returned " + resp.status);
  chrome.action.setBadgeText({ text: "" });
  return resp.json().catch(() => ({}));
}
