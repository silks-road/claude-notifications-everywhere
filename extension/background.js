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
