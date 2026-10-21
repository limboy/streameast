// NBA is generated from the Go code by `mygo generate`.
import { NBA, type Game, type ScheduledGame, type Team } from "./mygo";

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

let loading = false;
async function refresh() {
  if (loading) return;
  loading = true;
  $("#refresh").classList.add("busy");
  try {
    // Cards are for games to watch: live ones and today's. The site also
    // lists tomorrow's, which the schedule below shows instead.
    const today = dayKey(new Date());
    const games = (await NBA.games()).filter((g) => g.live || dayKey(new Date(g.start * 1000)) === today);
    $("#error").hidden = true;
    const list = $("#games");
    list.replaceChildren(...games.map(card));
    if (!games.length) list.append(el("p", "empty muted", "No games to watch right now. See the schedule below."));
    const live = games.filter((g) => g.live).length;
    $("#subtitle").textContent =
      `${games.length} game${games.length === 1 ? "" : "s"}` + (live ? ` · ${live} live` : "") +
      ` · updated ${new Date().toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })}`;
  } catch (err) {
    $("#error").textContent = `Couldn't load games: ${err instanceof Error ? err.message : err}. Check the domain.`;
    $("#error").hidden = false;
    $("#subtitle").textContent = "";
  } finally {
    loading = false;
    $("#refresh").classList.remove("busy");
  }
}

$("#refresh").addEventListener("click", refresh);
document.addEventListener("keydown", (e) => {
  if (e.metaKey && e.key === "r") {
    e.preventDefault();
    refresh();
  }
});

const CUSTOM = "custom";
const select = $<HTMLSelectElement>("#site-select");

function note(text: string, error = false) {
  const msg = $("#settings-msg");
  msg.textContent = text;
  msg.className = error ? "error" : "muted";
}

function option(value: string, label: string) {
  const o = el("option", "", label);
  o.value = value;
  return o;
}

// Fills the picker with the domains that work now, the current one
// selected even when it didn't answer.
async function loadDomains() {
  const current = (await NBA.settings()).site;
  select.disabled = true;
  select.replaceChildren(option(current, `${new URL(current).host} · checking…`));
  note("Checking which domains work…");
  const domains = await NBA.domains();
  const options = domains.map((d) =>
    option(d.site, `${d.host} · ${d.latency} ms · ${d.games} game${d.games === 1 ? "" : "s"}`),
  );
  if (!domains.some((d) => d.site === current)) {
    options.unshift(option(current, `${new URL(current).host} · not responding`));
  }
  select.replaceChildren(...options, option(CUSTOM, "Custom…"));
  select.value = current;
  select.disabled = false;
  $("#custom").hidden = true;
  note(
    domains.length
      ? `${domains.length} working domain${domains.length === 1 ? "" : "s"} found.`
      : "None of the known domains answered. Enter one that works for you.",
    !domains.length,
  );
}

async function useSite(site: string) {
  note("Checking…");
  try {
    const s = await NBA.setSite(site);
    note(`Using ${new URL(s.site).host}.`);
    refresh();
    return true;
  } catch (err) {
    note(err instanceof Error ? err.message : String(err), true);
    return false;
  }
}

$("#open-settings").addEventListener("click", () => {
  const form = $("#settings");
  form.hidden = !form.hidden;
  if (!form.hidden) loadDomains();
});

$("#recheck").addEventListener("click", loadDomains);

let previous = "";
select.addEventListener("focus", () => (previous = select.value));
select.addEventListener("change", async () => {
  if (select.value === CUSTOM) {
    $("#custom").hidden = false;
    $<HTMLInputElement>("#site").focus();
    return;
  }
  $("#custom").hidden = true;
  if (!(await useSite(select.value))) select.value = previous;
  previous = select.value;
});

$("#settings").addEventListener("submit", async (e) => {
  e.preventDefault();
  if (await useSite($<HTMLInputElement>("#site").value)) loadDomains();
});

refresh();
loadSchedule();
// Scores and live states change: keep them fresh while the window is open.
// The schedule is cached for 10 minutes in Go, and rolls over at midnight.
setInterval(() => {
  if (document.hidden) return;
  refresh();
  loadSchedule();
}, 60_000);
