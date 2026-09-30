// Read-only review page for rudder-cli dev listen. It polls the same query
// API as the CLI. Captured values are rendered with textContent only: they
// come from any app or web page that can reach the listener.
"use strict";

const API = "/_dev/v1/";
const POLL_MS = 1000;
const PAGE_LIMIT = 1000;
const MAX_EVENTS = 5000;

const state = {
  serverId: null,
  cursor: 0,
  events: [],
  summary: null,
  selected: null,
  search: "",
  failedOnly: false,
  source: "",
  requests: new Map(),
};

const $ = (id) => document.getElementById(id);

function el(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined && text !== null) node.textContent = String(text);
  if (className) node.className = className;
  return node;
}

async function getJSON(path) {
  const resp = await fetch(API + path, { headers: { Accept: "application/json" }, cache: "no-store" });
  if (!resp.ok) throw new Error(path + ": " + resp.status);
  return resp.json();
}

// failed follows the RudderStack Assistant: a beacon never reports a status,
// otherwise anything outside 2xx failed.
function failed(ev) {
  if (ev.transport === "beacon") return false;
  return !(ev.statusCode >= 200 && ev.statusCode < 300);
}

// matchesSearch is a case-insensitive substring match over the event name
// and every scalar value of the payload.
function matchesSearch(ev, needle) {
  if (!needle) return true;
  const stack = [ev.event, ev.message ?? ev.properties];
  const seen = new Set();
  while (stack.length > 0) {
    const v = stack.pop();
    if (v === null || v === undefined) continue;
    if (typeof v === "object") {
      if (seen.has(v)) continue;
      seen.add(v);
      for (const child of Array.isArray(v) ? v : Object.values(v)) stack.push(child);
      continue;
    }
    if (String(v).toLowerCase().includes(needle)) return true;
  }
  return false;
}

function visibleEvents() {
  const needle = state.search.trim().toLowerCase();
  return state.events.filter((ev) =>
    (!state.failedOnly || failed(ev)) &&
    (!state.source || ev.writeKey === state.source) &&
    matchesSearch(ev, needle));
}

function eventKey(ev) {
  return ev.seq + "." + ev.idx;
}

function clock(iso) {
  const d = new Date(iso);
  return isNaN(d) ? "-" : d.toLocaleTimeString();
}

function statusPill(code, isFailed) {
  return el("span", code, "pill " + (isFailed ? "bad" : "ok"));
}

function renderSummary() {
  const s = state.summary;
  if (!s) return;
  $("stat-requests").textContent = s.requests.total;
  $("stat-failed").textContent = s.requests.failed + " failed";
  $("stat-events").textContent = s.events.total;
  $("stat-types").textContent = Object.entries(s.events.byType).map(([k, v]) => k + " " + v).join(", ");
  $("stat-control").textContent = s.control.total;
  $("stat-control-sub").textContent = s.control.sourceConfig + " sourceConfig, " + s.control.preflight + " preflight";
  const keys = Object.keys(s.byWriteKey || {});
  $("stat-keys").textContent = keys.length;
  $("stat-sdks").textContent = Object.keys(s.bySource.bySdk || {}).join(", ");

  const items = s.diagnosis.map((d) => {
    const li = el("li", null, d.code === "all_accepted" ? "good" : "");
    li.append(el("span", d.code, "code"), el("span", d.message), el("code", d.next, "next"));
    return li;
  });
  $("diagnosis").replaceChildren(...items);

  const select = $("source");
  const current = select.value;
  const options = [el("option", "All write keys")];
  options[0].value = "";
  for (const key of keys.sort()) {
    const opt = el("option", key + " (" + s.byWriteKey[key].events + ")");
    opt.value = key;
    options.push(opt);
  }
  select.replaceChildren(...options);
  select.value = keys.includes(current) ? current : "";
}

function renderRows() {
  const rows = visibleEvents().slice().reverse();
  const trs = rows.map((ev) => {
    const tr = el("tr");
    tr.tabIndex = 0;
    tr.dataset.key = eventKey(ev);
    if (state.selected === eventKey(ev)) tr.className = "selected";
    const status = el("td");
    status.append(statusPill(ev.statusCode, failed(ev)));
    tr.append(el("td", clock(ev.receivedAt), "mono"), el("td", ev.type ?? "-"), el("td", ev.event ?? "-"), status,
      el("td", ev.writeKey || "-", "mono"));
    return tr;
  });
  $("rows").replaceChildren(...trs);
  $("empty").hidden = rows.length > 0;
  $("count").textContent = rows.length + " of " + state.events.length + " events";
}

function pretty(v) {
  return JSON.stringify(v ?? null, null, 2);
}

async function renderDetail() {
  const ev = state.events.find((e) => eventKey(e) === state.selected);
  $("detail-empty").hidden = Boolean(ev);
  $("detail-body").hidden = !ev;
  if (!ev) return;
  $("detail-title").textContent = ev.event ?? ev.type ?? "event";
  const pill = $("detail-status");
  pill.textContent = ev.statusCode;
  pill.className = "pill " + (failed(ev) ? "bad" : "ok");
  const facts = [["seq", eventKey(ev)], ["type", ev.type], ["received", ev.receivedAt], ["route", ev.route],
    ["write key", ev.writeKey], ["userId", ev.userId], ["anonymousId", ev.anonymousId], ["messageId", ev.messageId]];
  const nodes = [];
  for (const [k, v] of facts) {
    if (v === null || v === undefined || v === "") continue;
    nodes.push(el("dt", k), el("dd", v, "mono"));
  }
  $("detail-facts").replaceChildren(...nodes);
  $("detail-properties").textContent = pretty(ev.properties ?? ev.traits ?? {});

  const seq = ev.seq;
  if (!state.requests.has(seq)) {
    $("detail-request").textContent = "Loading request " + seq + "...";
    try {
      state.requests.set(seq, await getJSON("requests/" + seq + "?view=full&maxBytes=0"));
    } catch (err) {
      $("detail-request").textContent = "Could not load request " + seq + ": " + err.message;
      return;
    }
  }
  if (state.selected === eventKey(ev)) $("detail-request").textContent = pretty(state.requests.get(seq));
}

function reset(serverId) {
  state.serverId = serverId;
  state.cursor = 0;
  state.events = [];
  state.selected = null;
  state.requests.clear();
}

async function poll() {
  try {
    const counts = await getJSON("events?view=counts");
    if (counts.serverId !== state.serverId) reset(counts.serverId);
    state.summary = counts.summary;
    $("server").textContent = location.host + " · server " + counts.serverId;
    let more = true;
    while (more) {
      const page = await getJSON("events?view=full&maxBytes=0&limit=" + PAGE_LIMIT + "&since=" + state.cursor);
      if (page.serverId !== state.serverId) break;
      state.events.push(...page.events);
      state.cursor = page.cursor;
      more = page.hasMore;
    }
    if (state.events.length > MAX_EVENTS) state.events.splice(0, state.events.length - MAX_EVENTS);
    $("live").className = "dot on";
    renderSummary();
    renderRows();
    if (state.selected && !state.events.some((ev) => eventKey(ev) === state.selected)) {
      state.selected = null;
      renderDetail();
    }
  } catch (err) {
    $("live").className = "dot off";
    $("live").title = "Listener unreachable: " + err.message;
  } finally {
    setTimeout(poll, POLL_MS);
  }
}

function select(key) {
  state.selected = key;
  renderRows();
  renderDetail();
}

$("rows").addEventListener("click", (e) => {
  const tr = e.target.closest("tr");
  if (tr) select(tr.dataset.key);
});
$("rows").addEventListener("keydown", (e) => {
  const tr = e.target.closest("tr");
  if (!tr) return;
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    select(tr.dataset.key);
  } else if (e.key === "ArrowDown" || e.key === "ArrowUp") {
    e.preventDefault();
    const next = e.key === "ArrowDown" ? tr.nextElementSibling : tr.previousElementSibling;
    if (next) {
      next.focus();
      select(next.dataset.key);
    }
  }
});
$("search").addEventListener("input", (e) => {
  state.search = e.target.value;
  renderRows();
});
$("failed-only").addEventListener("change", (e) => {
  state.failedOnly = e.target.checked;
  renderRows();
});
$("source").addEventListener("change", (e) => {
  state.source = e.target.value;
  renderRows();
});
document.addEventListener("click", async (e) => {
  const button = e.target.closest("button.copy");
  if (!button) return;
  const text = $(button.dataset.copy === "request" ? "detail-request" : "detail-properties").textContent;
  try {
    await navigator.clipboard.writeText(text);
    button.textContent = "Copied";
  } catch {
    button.textContent = "Copy failed";
  }
  setTimeout(() => { button.textContent = "Copy"; }, 1200);
});

poll();
