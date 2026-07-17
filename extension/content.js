// content.js — runs on claude.ai. Detection happens at the network level in
// background.js; this script only answers "what does the page say" requests:
// the conversation title and the text of the last assistant message.

(function () {
  "use strict";

  function conversationTitle() {
    // Tab title is "<chat name> - Claude"; strip the suffix.
    return (document.title || "").replace(/\s*[-–—]\s*Claude\s*$/i, "").trim();
  }

  // Text of the last assistant message. Claude renders assistant turns in
  // .font-claude-message / prose containers; take the last non-empty one.
  // DOM may be virtualized, but the newest message is always rendered.
  function lastAssistantText() {
    const candidates = document.querySelectorAll(
      '.font-claude-message, [data-testid="user-message"] ~ div, [class*="prose"]'
    );
    let text = "";
    for (const el of candidates) {
      const t = (el.innerText || "").trim();
      if (t) text = t;
    }
    if (!text) {
      const main = document.querySelector("main");
      if (main) text = (main.innerText || "").trim();
    }
    return text.slice(0, 2000);
  }

  chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
    if (msg && msg.type === "get_last_message") {
      sendResponse({ title: conversationTitle(), lastMessage: lastAssistantText() });
    }
  });
})();
