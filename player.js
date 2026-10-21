// Runs in the game's page before its own scripts (PageOptions.PreloadScript).
// window.__NBA_PLAYER__ = { path } comes from player.go.
(() => {
  const cfg = window.__NBA_PLAYER__ || {};

  // Popups, pop-unders and new tabs are ads.
  window.open = () => null;

  // Third-party scripts are ads and trackers: neuter them before they run,
  // as the parser inserts them. The site's own scripts stay.
  const blocked = (src) => {
    try {
      return new URL(src, location.href).host !== location.host;
    } catch {
      return false;
    }
  };
  const block = (node) => {
    if (node.tagName === "SCRIPT" && node.src && blocked(node.src)) {
      node.type = "javascript/blocked";
      node.remove();
    }
  };
  new MutationObserver((records) => {
    for (const r of records) for (const n of r.addedNodes) block(n);
  }).observe(document.documentElement, { childList: true, subtree: true });

  // Show only the player: everything else is hidden, and hidden elements
  // take no clicks, so the site's click-catching overlays are gone too.
  const css = `
    html, body { background: #000 !important; overflow: hidden !important; }
    body * { visibility: hidden !important; }
    #iframe, #iframe * { visibility: visible !important; }
    #iframe {
      position: fixed !important; inset: 0 !important;
      width: 100vw !important; height: 100vh !important;
      max-width: none !important; max-height: none !important;
      margin: 0 !important; border: 0 !important; z-index: 2147483000 !important;
    }
    #nba-hot { position: fixed; top: 0; right: 0; width: 240px; height: 56px; z-index: 2147483646; visibility: visible !important; }
    #nba-bar, #nba-bar *, #nba-msg, #nba-msg * { visibility: visible !important; }
    #nba-bar {
      --surface: rgb(255 255 255 / .82); --ink: #1d2129; --hover: rgb(0 0 0 / .07); --edge: rgb(0 0 0 / .1);
      position: fixed; top: 12px; right: 12px; z-index: 2147483647;
      display: flex; align-items: center; gap: 2px; padding: 3px;
      font: 600 12px/1 -apple-system, system-ui, sans-serif; color: var(--ink);
      background: var(--surface); border: 1px solid var(--edge); border-radius: 10px;
      -webkit-backdrop-filter: blur(16px) saturate(1.6); backdrop-filter: blur(16px) saturate(1.6);
      box-shadow: 0 4px 16px rgb(0 0 0 / .25);
      transition: opacity .2s, transform .2s;
    }
    @media (prefers-color-scheme: dark) {
      #nba-bar { --surface: rgb(28 31 38 / .78); --ink: #eef0f4; --hover: rgb(255 255 255 / .1); --edge: rgb(255 255 255 / .12); }
    }
    #nba-bar.hide { opacity: 0; transform: translateY(-6px); pointer-events: none; }
    #nba-bar button {
      font: inherit; color: inherit; cursor: pointer; background: none; border: 0;
      min-width: 28px; height: 26px; padding: 0 8px; border-radius: 7px;
      display: grid; place-items: center;
    }
    #nba-bar button:hover { background: var(--hover); }
    #nba-bar button.on { background: #f97316; color: #fff; }
    #nba-bar .sep { width: 1px; height: 16px; margin: 0 3px; background: var(--edge); }
    #nba-bar svg { width: 14px; height: 14px; }
    #nba-msg {
      position: fixed; inset: 0; display: grid; place-content: center; gap: 8px; text-align: center;
      font: 15px -apple-system, system-ui, sans-serif; color: #c8ccd4; z-index: 2147482000;
    }
    #nba-msg b { color: #fff; font-size: 18px; }
  `;
  const style = document.createElement("style");
  style.textContent = css;
  document.documentElement.appendChild(style);

  document.addEventListener("DOMContentLoaded", () => {
    // The page may have moved its sheet; keep ours last.
    document.head.appendChild(style);

    // The free servers, from the site's own list. Premium ones need an
    // account.
    const servers = [...document.querySelectorAll(".se-stream:not(.is-pro):not(.se-stream--share) a.se-stream__link")]
      .map((a) => ({ name: a.textContent.trim(), href: a.getAttribute("href") }))
      .filter((s) => s.href && s.href.startsWith(cfg.path));

    // A small picker in the top-right corner: the servers by number, then
    // reload. The window title already names the game.
    const bar = document.createElement("div");
    bar.id = "nba-bar";
    for (const s of servers) {
      const b = document.createElement("button");
      b.textContent = s.name.replace(/^Server\s*/i, "");
      b.title = s.name;
      if (new URL(s.href, location.href).pathname.replace(/\/$/, "") === location.pathname.replace(/\/$/, "")) b.className = "on";
      b.onclick = () => location.assign(s.href);
      bar.append(b);
    }
    const sep = document.createElement("span");
    sep.className = "sep";
    const reload = document.createElement("button");
    reload.title = "Reload this server";
    reload.innerHTML =
      '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M13.5 8a5.5 5.5 0 1 1-1.6-3.9"/><path d="M13.5 2.5v3h-3"/></svg>';
    reload.onclick = () => location.reload();
    bar.append(sep, reload);

    const hot = document.createElement("div");
    hot.id = "nba-hot";
    document.body.append(hot, bar);

    // The picker hides while the video plays; moving the pointer to the
    // top-right corner, or over the page outside the video, brings it back.
    let timer;
    const show = () => {
      bar.classList.remove("hide");
      clearTimeout(timer);
      timer = setTimeout(() => {
        if (!bar.matches(":hover")) bar.classList.add("hide");
      }, 2500);
    };
    hot.addEventListener("mouseenter", show);
    bar.addEventListener("mouseleave", show);
    document.addEventListener("mousemove", show);
    show();

    if (!document.querySelector("#iframe")) {
      const msg = document.createElement("div");
      msg.id = "nba-msg";
      const head = document.createElement("b");
      head.textContent = "No stream here yet";
      const sub = document.createElement("span");
      sub.textContent = "Streams usually start shortly before tip-off. Try another server, or Reload.";
      msg.append(head, sub);
      document.body.append(msg);
    }
  });
})();
