// NBA is generated from the Go code by `mygo generate`.
import { NBA, type Domain, type Game, type GameDetail, type Leader, type ScheduledGame, type Team } from "./mygo";

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
  const state = game.live ? " live" : game.final ? " final" : "";
  const c = el("button", "game" + state);
  c.title = game.final
    ? `Final: ${game.away.name} ${game.away.score} – ${game.home.score} ${game.home.name}`
    : `Watch ${game.away.name} @ ${game.home.name}`;
  const head = el("div", "head");
  if (game.live) {
    head.append(el("span", "pill", "LIVE"), el("span", "status", game.status));
  } else if (game.final) {
    head.append(el("span", "pill final", "FINAL"));
    // "Final/OT" is worth saying; plain "Final" would just repeat the pill.
    if (game.status && game.status !== "Final") head.append(el("span", "status", game.status));
  } else {
    head.append(el("span", "status", startTime(game)));
  }
  c.append(head, team(game.away, "away"), team(game.home, "home"));
  c.onclick = () => (game.final ? openDetail(game) : NBA.watch(game));
  return c;
}

const SCHEDULE_DAYS = 3;

const detailOverlay = $<HTMLDialogElement>("#detail");
const detailBody = $<HTMLDivElement>("#detail-body");

const gameDate = (start: number) =>
  new Date(start * 1000).toLocaleString([], {
    weekday: "short",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });

function detailSide(t: Team, record: string, side: "away" | "home") {
  const box = el("div", `dside ${side}`);
  const info = el("div", "dinfo");
  const name = el("span", "dname", t.short || t.name);
  name.title = t.name;
  info.append(name, el("span", "drecord muted", record));
  box.append(logo(t, "logo big"), info, el("span", "dscore", t.score));
  return box;
}

function lineScore(d: GameDetail) {
  const table = el("table", "linescore");
  const head = el("thead");
  const headRow = el("tr");
  headRow.append(el("th", "", ""));
  for (const p of d.periods) headRow.append(el("th", "", p));
  headRow.append(el("th", "total", "T"));
  head.append(headRow);

  const body = el("tbody");
  const rows: Array<[Team, string[]]> = [
    [d.away, d.awayLine],
    [d.home, d.homeLine],
  ];
  for (const [t, line] of rows) {
    const row = el("tr");
    const name = el("td", "tname");
    const inner = el("span", "tname-inner");
    inner.append(logo(t, "logo tiny"), el("span", "", t.short || t.name));
    name.append(inner);
    row.append(name);
    for (let i = 0; i < d.periods.length; i++) row.append(el("td", "", line[i] ?? "–"));
    row.append(el("td", "total", t.score));
    body.append(row);
  }
  table.append(head, body);
  return table;
}

// ESPN's team stats: the values either side, a bar under each, longest to
// the better number.
function statBars(d: GameDetail) {
  const box = el("div", "stats");
  for (const s of d.stats) {
    const row = el("div", "stat");
    const values = el("div", "stat-values");
    values.append(el("span", "sv away", s.away), el("span", "sl", s.label), el("span", "sv home", s.home));

    const max = Math.max(s.awayValue, s.homeValue, 1);
    const bar = el("div", "stat-bar");
    for (const side of ["away", "home"] as const) {
      const half = el("div", `half ${side}`);
      const fill = el("span", `sb ${side}`);
      fill.style.width = `${((side === "away" ? s.awayValue : s.homeValue) / max) * 100}%`;
      half.append(fill);
      bar.append(half);
    }
    row.append(values, bar);
    box.append(row);
  }
  return box;
}

function leaderBox(t: Team, list: Leader[]) {
  const box = el("div", "leader-team");
  const head = el("div", "dlhead");
  head.append(logo(t, "logo small"), el("span", "", t.short || t.name));
  box.append(head);
  for (const l of list) {
    const row = el("div", "leader");
    if (l.headshot) {
      const img = el("img", "headshot");
      img.src = l.headshot;
      img.alt = "";
      img.onerror = () => img.classList.add("missing");
      row.append(img);
    }
    const info = el("div", "linfo");
    info.append(el("span", "lname", l.name), el("span", "ldetail muted", l.detail));
    const stat = el("div", "lstat");
    stat.append(el("span", "lval", l.value), el("span", "lcat muted", l.category));
    row.append(info, stat);
    box.append(row);
  }
  return box;
}

function renderDetail(d: GameDetail) {
  const root = el("div", "detail-content");
  if (d.away.color) root.style.setProperty("--away-color", d.away.color);
  if (d.home.color) root.style.setProperty("--home-color", d.home.color);

  const meta = el("div", "dmeta");
  meta.append(el("span", "dstatus", d.status || "Final"));
  if (d.start) meta.append(el("span", "dwhen", gameDate(d.start)));
  if (d.venue) meta.append(el("span", "dvenue", d.venue));

  const head = el("div", "detail-head");
  head.append(detailSide(d.away, d.awayRecord, "away"), meta, detailSide(d.home, d.homeRecord, "home"));
  root.append(head);

  if (d.periods.length) root.append(lineScore(d));

  if (d.stats.length) {
    root.append(el("h4", "", "Team Stats"), statBars(d));
  }

  if (d.awayLeaders.length || d.homeLeaders.length) {
    root.append(el("h4", "", "Top Performers"));
    const cols = el("div", "leader-cols");
    if (d.awayLeaders.length) cols.append(leaderBox(d.away, d.awayLeaders));
    if (d.homeLeaders.length) cols.append(leaderBox(d.home, d.homeLeaders));
    root.append(cols);
  }

  if (!d.periods.length && !d.stats.length && !d.awayLeaders.length) {
    root.append(el("p", "muted", "ESPN hasn't published the details of this game yet."));
  }
  detailBody.replaceChildren(root);
}

let detailRequest = 0;

async function openDetail(game: Game) {
  const request = ++detailRequest;
  if (!detailOverlay.open) detailOverlay.showModal();
  document.body.classList.add("detail-open");
  detailBody.replaceChildren(el("p", "muted loading", "Loading game details…"));
  try {
    const d = await NBA.detail(game);
    if (request !== detailRequest) return;
    renderDetail(d);
  } catch (err) {
    if (request !== detailRequest) return;
    detailBody.replaceChildren(
      el("p", "error", `Couldn't load the details: ${err instanceof Error ? err.message : err}`),
    );
  }
}

function closeDetail() {
  detailRequest++;
  detailOverlay.close();
  document.body.classList.remove("detail-open");
  detailBody.replaceChildren();
}

$("#detail-close").addEventListener("click", closeDetail);
detailOverlay.addEventListener("click", (e) => {
  if (e.target === detailOverlay) closeDetail();
});
detailOverlay.addEventListener("cancel", (e) => {
  e.preventDefault();
  closeDetail();
});

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
    // Cards are today's games: live ones, those still to come, and the ones
    // that ended. The site also lists tomorrow's, which the schedule below
    // shows instead.
    const today = dayKey(new Date());
    const games = (await NBA.games()).filter(
      (g) => g.live || dayKey(new Date(g.start * 1000)) === today,
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
