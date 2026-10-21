// NBA is generated from the Go code by `mygo generate`.
import { NBA, type Game, type Team } from "./mygo";

const $ = <T extends HTMLElement>(selector: string) => document.querySelector<T>(selector)!;

const el = <K extends keyof HTMLElementTagNameMap>(tag: K, className = "", text = "") => {
  const e = document.createElement(tag);
  if (className) e.className = className;
  if (text) e.textContent = text;
  return e;
};

const startTime = (game: Game) =>
  new Date(game.start * 1000).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });

function team(t: Team, side: string) {
  const row = el("div", `team ${side}`);
  const logo = el("img", "logo");
  logo.alt = "";
  if (t.logo) logo.src = t.logo;
  logo.onerror = () => logo.classList.add("missing");
  row.append(logo, el("span", "name", t.short || t.name), el("span", "score", t.score));
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

let loading = false;
async function refresh() {
  if (loading) return;
  loading = true;
  $("#refresh").classList.add("busy");
  try {
    const games = await NBA.games();
    $("#error").hidden = true;
    const list = $("#games");
    list.replaceChildren(...games.map(card));
    if (!games.length) list.append(el("p", "empty muted", "No NBA games on the site right now."));
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
// Scores and live states change: keep them fresh while the window is open.
setInterval(() => {
  if (!document.hidden) refresh();
}, 60_000);
