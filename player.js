// Runs in the game's page before its own scripts (PageOptions.PreloadScript),
// and, since it clears the ads of the frames the player comes in, in every
// iframe too (PageOptions.PreloadAllFrames). window.__NBA_PLAYER__ =
// { title, path } comes from player.go, into the top frame alone.
(() => {
  const cfg = window.__NBA_PLAYER__ || {};

  // Popups, pop-unders and new tabs are ads.
  window.open = () => null;

  if (window !== window.top) {
    // The player is an iframe two origins deep (/iframe → wikisport →
    // the player's own host), and the ads there lay a transparent div over
    // the whole frame, at the top z-index, which eats the click that starts
    // playback: each click on the video goes to it instead, and with popups
    // blocked nothing ever comes of it. Nothing else here is ours in a
    // frame: the frame's scripts include the player's own, so they stay.
    const playerSide = (el) =>
      el.tagName === "VIDEO" ||
      (el.closest && el.closest("#jwplayer, .jwplayer, #player-container, #unmuteBtn, video"));
    const clear = (el) => {
      // The layer is an empty div: the check is cheap on everything else,
      // which matters in the chat frame, with thousands of nodes.
      if (!(el instanceof HTMLElement) || el.childElementCount || el.textContent.trim()) return;
      if (playerSide(el)) return;
      const s = getComputedStyle(el);
      if (s.position !== "fixed" || s.display === "none" || s.visibility === "hidden") return;
      if (+s.zIndex < 1e5) return;
      if (!/^(transparent|rgba\(0, 0, 0, 0\))$/.test(s.backgroundColor)) return;
      const b = el.getBoundingClientRect();
      if (b.width < innerWidth * 0.95 || b.height < innerHeight * 0.95) return;
      // Not removed: the ad script would put another one back, and takes
      // the player's own cleanup with it. Clicks go through it instead.
      el.style.setProperty("pointer-events", "none", "important");
    };
    const sweep = () => {
      if (document.body) for (const el of document.body.getElementsByTagName("*")) clear(el);
    };
    // The layers come with the ads, and are styled after they are inserted:
    // look at what arrives, and again as the page settles.
    new MutationObserver((records) => {
      for (const r of records) {
        for (const n of r.addedNodes) if (n.nodeType === 1) clear(n);
      }
    }).observe(document.documentElement, { childList: true, subtree: true });
    for (const t of [0, 500, 1000, 2000, 4000, 8000, 15000]) setTimeout(sweep, t);
    return;
  }

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
    :root { --nba-bar: 40px; }
    :root.nba-fullscreen { --nba-bar: 0px; }
    #iframe {
      position: fixed !important; inset: var(--nba-bar) 0 0 0 !important;
      width: 100vw !important; height: calc(100vh - var(--nba-bar)) !important;
      max-width: none !important; max-height: none !important;
      margin: 0 !important; border: 0 !important; z-index: 2147483000 !important;
    }
    #nba-bar, #nba-bar *, #nba-msg, #nba-msg * { visibility: visible !important; }
    #nba-bar {
      --surface: #ececee; --ink: #1d2129; --muted: #6b7280; --hover: rgb(0 0 0 / .07); --edge: rgb(0 0 0 / .1);
      position: fixed; top: 0; left: 0; right: 0; height: var(--nba-bar); z-index: 2147483647;
      display: flex; align-items: center; gap: 2px; box-sizing: border-box;
      padding: 0 8px 0 calc(var(--mygo-titlebar-inset-left, 0px) + 8px);
      font: 600 12px/1 -apple-system, system-ui, sans-serif; color: var(--ink);
      background: var(--surface); border-bottom: 1px solid var(--edge);
      --app-region: drag; -webkit-user-select: none; user-select: none; overflow: hidden;
    }
    :root.nba-fullscreen #nba-bar { display: none; }
    @media (prefers-color-scheme: dark) {
      #nba-bar { --surface: #1c1f26; --ink: #eef0f4; --muted: #9aa0a6; --hover: rgb(255 255 255 / .1); --edge: rgb(255 255 255 / .1); }
    }
    #nba-bar .t {
      margin-right: auto; padding-right: 12px; color: var(--muted); font-weight: 500; font-size: 13px;
      white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    }
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
      position: fixed; inset: var(--nba-bar) 0 0 0; display: grid; place-content: center; gap: 8px; text-align: center;
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

    // The free servers, from StreamEast's list.
    const seServers = [...document.querySelectorAll(".se-stream:not(.is-pro):not(.se-stream--share) a.se-stream__link")]
      .map((a) => ({ name: a.textContent.trim(), href: a.getAttribute("href") }))
      .filter((s) => s.href && s.href.startsWith(cfg.path));

    // A different game may not offer the server remembered from the last
    // one. Fall back once to server 1, including on an empty error page.
    const pagePath = (href) => new URL(href, location.href).pathname.replace(/\/$/, "");
    const currentPath = pagePath(location.href);
    const currentServer = cfg.path && location.pathname.startsWith(cfg.path)
      ? Number(location.pathname.slice(cfg.path.length).replace(/\/$/, ""))
      : 0;
    if (currentServer > 1 && !seServers.some((s) => pagePath(s.href) === currentPath)) {
      location.replace(cfg.path + "1");
      return;
    }

    // The title bar, above the video: the game, then the servers by number
    // and reload on the right.
    const bar = document.createElement("div");
    bar.id = "nba-bar";
    const title = document.createElement("span");
    title.className = "t";
    title.textContent = cfg.title || "";
    bar.append(title);

    for (const s of seServers) {
      const b = document.createElement("button");
      b.textContent = s.name.replace(/^Server\s*/i, "");
      b.title = s.name;
      if (pagePath(s.href) === currentPath) b.className = "on";
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

    document.body.append(bar);

    // In full screen the window controls hide and MyGo reports a title bar
    // of 0px: the bar goes too, and the video fills the screen.
    const fit = () => {
      const h = getComputedStyle(document.documentElement).getPropertyValue("--mygo-titlebar-height").trim();
      document.documentElement.classList.toggle("nba-fullscreen", h === "0px");
    };
    addEventListener("resize", fit);
    new MutationObserver(fit).observe(document.documentElement, { attributes: true, attributeFilter: ["style"] });
    fit();

    const checkStream = () => document.querySelector("#iframe");

    if (!checkStream()) {
      const msg = document.createElement("div");
      msg.id = "nba-msg";
      const head = document.createElement("b");
      head.textContent = "No stream here yet";
      const sub = document.createElement("span");
      sub.textContent = "Streams usually start shortly before tip-off. Try another server, or Reload.";
      msg.append(head, sub);
      document.body.append(msg);

      const observer = new MutationObserver(() => {
        if (checkStream()) {
          msg.remove();
          observer.disconnect();
        }
      });
      observer.observe(document.body, { childList: true, subtree: true });
    }
  });
})();
