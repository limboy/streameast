// NBA is generated from the Go code by `mygo generate`.
import { NBA, type Domain, type Game, type ScheduledGame, type Team } from "./mygo";

const $ = <T extends HTMLElement>(selector: string) => document.querySelector<T>(selector)!;

const el = <K extends keyof HTMLElementTagNameMap>(tag: K, className = "", text = "") => {
  const e = document.createElement(tag);
  if (className) e.className = className;
  if (text) e.textContent = text;
  return e;
};

const startTime = (game: Game) =>
  new Date(game.start * 1000).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });

// The team's logo, in its light or dark version as the system's theme is.
function logo(t: Team, className: string) {
  const picture = el("picture");
  if (t.logoDark && t.logoDark !== t.logo) {
    const dark = el("source");
    dark.media = "(prefers-color-scheme: dark)";
    dark.srcset = t.logoDark;
    picture.append(dark);
  }
  const img = el("img", className);
  img.alt = "";
  if (t.logo) img.src = t.logo;
  img.onerror = () => img.classList.add("missing");
  picture.append(img);
  return picture;
}

function team(t: Team, side: string) {
  const row = el("div", `team ${side}`);
  row.append(logo(t, "logo"), el("span", "name", t.short || t.name), el("span", "score", t.score));
  return row;
}

function card(game: Game) {
  const c = el("button", "game" + (game.live ? " live" : ""));
  c.title = `Watch ${game.away.name} @ ${game.home.name}`;
  const head = el("div", "head");
  if (game.live) {
    head.append(el("span", "pill", "LIVE"), el("span", "status", game.status));
  } else {
    head.append(el("span", "status", startTime(game)));
  }
  c.append(head, team(game.away, "away"), team(game.home, "home"));
  c.onclick = () => NBA.watch(game);
  return c;
}

const SCHEDULE_DAYS = 3;

const dayKey = (d: Date) => `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;

function scheduleRow(game: ScheduledGame) {
  const row = el("div", "match");
  const side = (t: Team) => {
    const s = el("span", "side");
    s.append(logo(t, "logo small"), el("span", "", t.short || t.name));
    s.title = t.name;
    return s;
  };
  row.append(
    el("span", "time", new Date(game.start * 1000).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })),
    side(game.away),
    el("span", "at muted", "@"),
    side(game.home),
    el("span", "tv muted", game.broadcasts.join(" · ")),
  );
  return row;
}

// The games of the next SCHEDULE_DAYS days, by the user's own dates.
async function loadSchedule() {
  const days = $("#days");
  let games: ScheduledGame[];
  try {
    games = await NBA.schedule();
  } catch (err) {
    days.replaceChildren(el("p", "error", `Couldn't load the schedule: ${err instanceof Error ? err.message : err}`));
    return;
  }
  const groups: HTMLElement[] = [];
  const today = new Date();
  for (let i = 1; i <= SCHEDULE_DAYS; i++) {
    const day = new Date(today.getFullYear(), today.getMonth(), today.getDate() + i);
    const matches = games.filter((g) => dayKey(new Date(g.start * 1000)) === dayKey(day));
    const date = day.toLocaleDateString([], { weekday: "short", month: "short", day: "numeric" });
    const group = el("div", "day");
    const head = el("h3");
    head.append(el("span", "", i === 1 ? "Tomorrow" : day.toLocaleDateString([], { weekday: "long" })));
    head.append(el("span", "muted", ` · ${date}`));
    group.append(head);
    if (matches.length) group.append(...matches.map(scheduleRow));
    else group.append(el("p", "muted none", "No games"));
    groups.push(group);
  }
  days.replaceChildren(...groups);
}

const refreshBtn = $<HTMLButtonElement>("#refresh");
const refreshTooltip = $<HTMLSpanElement>("#refresh-tooltip");
const sourceSelect = $<HTMLSelectElement>("#source-select");

let lastRefreshed: Date | null = null;
let loadingRefreshes = 0;
let gamesLoading = false;

function updateRefreshTooltip() {
  if (gamesLoading || loadingRefreshes > 0) {
    refreshTooltip.textContent = "Refreshing…";
    return;
  }
  if (!lastRefreshed) {
    refreshTooltip.textContent = "Not refreshed yet";
    return;
  }
  const timeStr = lastRefreshed.toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
  });
  refreshTooltip.textContent = `Updated ${timeStr}`;
}

function setBusy(busy: boolean) {
  if (busy) {
    loadingRefreshes++;
    refreshBtn.classList.add("busy");
  } else {
    loadingRefreshes = Math.max(0, loadingRefreshes - 1);
    if (loadingRefreshes === 0) {
      refreshBtn.classList.remove("busy");
    }
  }
  updateRefreshTooltip();
}

async function refresh() {
  if (gamesLoading) return;
  gamesLoading = true;
  setBusy(true);
  try {
    // Cards are for games to watch: live ones and today's (excluding finished games).
    // The site also lists tomorrow's, which the schedule below shows instead.
    const today = dayKey(new Date());
    const games = (await NBA.games()).filter(
      (g) => g.status !== "Final" && (g.live || dayKey(new Date(g.start * 1000)) === today),
    );
    $("#error").hidden = true;
    const list = $("#games");
    list.replaceChildren(...games.map(card));
    // With nothing to watch, the schedule moves up.
    list.hidden = !games.length;
    lastRefreshed = new Date();
  } catch (err) {
    $("#error").textContent = `Couldn't load games: ${err instanceof Error ? err.message : err}. Check or switch the source.`;
    $("#error").hidden = false;
  } finally {
    gamesLoading = false;
    setBusy(false);
  }
}

interface SourceItem {
  site: string;
  host: string;
  latency?: number;
}

const DEFAULT_SOURCES: SourceItem[] = [
  { site: "https://v2.streameast.ga", host: "v2.streameast.ga" },
  { site: "https://v2.streameast.ch", host: "v2.streameast.ch" },
  { site: "https://v2.thestreameast.fun", host: "v2.thestreameast.fun" },
];

let activeSite = "https://v2.streameast.ga";
let currentSources: SourceItem[] = [...DEFAULT_SOURCES];

function sourceLabel(s: SourceItem): string {
  if (s.latency !== undefined) {
    return `${s.host} · ${s.latency} ms`;
  }
  return s.host;
}

function renderSourceOptions(sources: SourceItem[], currentSite: string) {
  const existingOptions = Array.from(sourceSelect.options);
  const sameKeys =
    existingOptions.length === sources.length &&
    existingOptions.every((opt, i) => sources[i] && opt.value === sources[i]!.site);

  if (sameKeys) {
    sources.forEach((s, i) => {
      const opt = existingOptions[i];
      if (opt) opt.textContent = sourceLabel(s);
    });
  } else {
    sourceSelect.innerHTML = "";
    for (const s of sources) {
      const opt = document.createElement("option");
      opt.value = s.site;
      opt.textContent = sourceLabel(s);
      sourceSelect.appendChild(opt);
    }
  }

  for (const opt of Array.from(sourceSelect.options)) {
    try {
      if (opt.value === currentSite || new URL(opt.value).host === new URL(currentSite).host) {
        opt.selected = true;
        break;
      }
    } catch {
      if (opt.value === currentSite) {
        opt.selected = true;
        break;
      }
    }
  }
}

let testingSources = false;
async function updateSources() {
  if (testingSources) return;
  testingSources = true;
  setBusy(true);
  try {
    const domains = await NBA.domains();
    if (domains && domains.length) {
      const merged: SourceItem[] = DEFAULT_SOURCES.map((def) => {
        const live = domains.find((d) => d.site === def.site || d.host === def.host);
        if (live) {
          return {
            site: live.site,
            host: live.host,
            latency: live.latency,
          };
        }
        return def;
      });

      for (const d of domains) {
        if (!merged.some((m) => m.site === d.site || m.host === d.host)) {
          merged.push({
            site: d.site,
            host: d.host,
            latency: d.latency,
          });
        }
      }

      currentSources = merged;
      renderSourceOptions(currentSources, activeSite);
    }
  } catch {
    // ignore
  } finally {
    testingSources = false;
    setBusy(false);
  }
}

async function useSite(site: string) {
  try {
    const s = await NBA.setSite(site);
    activeSite = s.site;
    $("#error").hidden = true;
    renderSourceOptions(currentSources, activeSite);
    await refresh();
    return true;
  } catch (err) {
    $("#error").textContent = `Couldn't connect to ${site}: ${err instanceof Error ? err.message : err}`;
    $("#error").hidden = false;
    return false;
  }
}

sourceSelect.addEventListener("change", async () => {
  const selected = sourceSelect.value;
  if (!selected || selected === activeSite) return;
  const ok = await useSite(selected);
  if (!ok) {
    renderSourceOptions(currentSources, activeSite);
  }
});

async function triggerRefresh() {
  await Promise.allSettled([refresh(), updateSources()]);
}

refreshBtn.addEventListener("click", () => {
  triggerRefresh();
});

refreshBtn.addEventListener("mouseenter", updateRefreshTooltip);

document.addEventListener("keydown", (e) => {
  if (e.metaKey && e.key === "r") {
    e.preventDefault();
    triggerRefresh();
  }
});

async function initSources() {
  try {
    const settings = await NBA.settings();
    if (settings && settings.site && !settings.site.includes("streamseaste")) {
      activeSite = settings.site;
    }
  } catch {}
  renderSourceOptions(currentSources, activeSite);
  updateSources();
}

refresh();
loadSchedule();
initSources();

// Scores and live states change: keep them fresh while the window is open.
// The schedule is cached for 10 minutes in Go, and rolls over at midnight.
setInterval(() => {
  if (document.hidden) return;
  refresh();
  loadSchedule();
}, 60_000);
