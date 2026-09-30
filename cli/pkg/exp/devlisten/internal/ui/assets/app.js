// Read-only review page for the local capture server. It reads the same
// query API as the command line and adds no server behaviour. Captured
// values are rendered with textContent only: any app or web page that can
// reach the listener can put text in them.
"use strict";

const API = "/_dev/v1/";
const PAGE_LIMIT = 1000;
const LIVE_WAIT = "5s";
const MAX_EVENTS = 5000;

// URL query keys that hold the view; the page URL is shareable.
const STATE_KEYS = ["tab", "q", "event", "type", "userId", "anonymousId", "statusCode", "writeKey", "kind",
  "failed", "fields", "since", "paused"];

const state = {
  tab: "events",
  q: "",
  event: "",
  type: "",
  userId: "",
  anonymousId: "",
  statusCode: "",
  writeKey: null,
  kind: "all",
  failed: false,
  fields: "",
  since: 0,
  paused: false,

  serverId: null,
  cursor: 0,
  envelope: null,
  events: [],
  rejected: [],
  requests: [],
  selected: null,
  detailTab: "compact",
  requestCache: new Map(),
  generation: 0,
  controller: null,
};

const $ = (id) => document.getElementById(id);

function el(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined && text !== null) node.textContent = String(text);
  if (className) node.className = className;
  return node;
}

function button(text, onClick, className) {
  const b = el("button", text, className || "btn");
  b.type = "button";
  b.addEventListener("click", onClick);
  return b;
}

// ---- URL state --------------------------------------------------------

function readURL() {
  const p = new URLSearchParams(location.search);
  for (const key of STATE_KEYS) {
    if (!p.has(key)) continue;
    const v = p.get(key);
    if (key === "failed" || key === "paused") state[key] = v === "1";
    else if (key === "since") state.since = Number(v) || 0;
    else state[key] = v;
  }
}

function writeURL() {
  const p = new URLSearchParams();
  for (const key of STATE_KEYS) {
    const v = state[key];
    if (v === "" || v === null || v === false || v === 0 || (key === "tab" && v === "events") ||
      (key === "kind" && v === "all")) continue;
    p.set(key, v === true ? "1" : String(v));
  }
  const qs = p.toString();
  history.replaceState(null, "", qs ? "?" + qs : location.pathname);
}

// ---- API --------------------------------------------------------------

function listValues(text) {
  return text.split(",").map((s) => s.trim()).filter(Boolean);
}

// sharedParams are the filters the events and requests routes share.
function sharedParams(p) {
  for (const code of listValues(state.statusCode)) p.append("statusCode", code);
  if (state.writeKey !== null) p.append("writeKey", state.writeKey);
  if (state.serverId) p.set("serverId", state.serverId);
}

function eventsQuery(since, wait) {
  const p = new URLSearchParams({ view: "full", maxBytes: "0", limit: String(PAGE_LIMIT), since: String(since) });
  for (const name of listValues(state.event)) p.append("event", name);
  if (state.type) p.set("type", state.type);
  if (state.userId) p.set("userId", state.userId);
  if (state.anonymousId) p.set("anonymousId", state.anonymousId);
  sharedParams(p);
  if (wait) {
    p.set("wait", wait);
    p.set("min", "1");
  }
  return "events?" + p.toString();
}

function requestsQuery(kind, failedOnly) {
  const p = new URLSearchParams({ view: "full", maxBytes: "0", limit: String(PAGE_LIMIT), since: String(state.since),
    kind });
  if (failedOnly) p.set("failed", "true");
  sharedParams(p);
  return "requests?" + p.toString();
}

async function getJSON(path, signal) {
  const resp = await fetch(API + path, { headers: { Accept: "application/json" }, cache: "no-store", signal });
  const body = await resp.json();
  if (!resp.ok) {
    const err = new Error(body?.error?.message || String(resp.status));
    err.code = body?.error?.code;
    throw err;
  }
  return body;
}

// ---- model ------------------------------------------------------------

// failed follows the RudderStack Assistant: a beacon never reports a
// status, otherwise anything outside 2xx failed.
function failed(item) {
  if (item.transport === "beacon") return false;
  return !(item.statusCode >= 200 && item.statusCode < 300);
}

function matchesSearch(values, needle) {
  if (!needle) return true;
  const stack = [...values];
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

function keyOf(item) {
  return item.kind === "rejected" ? "r" + item.seq : item.seq + "." + item.idx;
}

// seqLabel is the request seq, with the event index only when the request
// carried several events.
function seqLabel(item) {
  const siblings = state.events.filter((ev) => ev.seq === item.seq).length;
  return siblings > 1 ? item.seq + "/" + item.idx : String(item.seq);
}

// label names an event that has no event name: the page or screen name, or
// the identify traits.
function label(ev) {
  if (ev.event) return ev.event;
  const m = ev.message || {};
  if (ev.type === "page" || ev.type === "screen") return m.name || m.properties?.name || m.properties?.path || ev.type;
  if (ev.type === "identify") return ev.userId || Object.keys(ev.traits || {}).join(", ") || "identify";
  return ev.type || "-";
}

// rejectedRow turns a failed request without events into a list row, so
// Failed only shows every refusal.
function rejectedRow(rec) {
  return { kind: "rejected", seq: rec.seq, idx: 0, receivedAt: rec.receivedAt, type: "rejected",
    event: rec.rejection?.reason || rec.route, statusCode: rec.statusCode, writeKey: rec.writeKey,
    transport: rec.transport, route: rec.route };
}

function eventRows() {
  const needle = state.q.trim().toLowerCase();
  const rows = state.events.concat(state.rejected.filter((r) => r.events.length === 0).map(rejectedRow));
  return rows
    .filter((ev) => !state.failed || failed(ev))
    .filter((ev) => matchesSearch([label(ev), ev.message ?? ev.properties], needle))
    .sort((a, b) => b.seq - a.seq || b.idx - a.idx);
}

function requestRows() {
  const needle = state.q.trim().toLowerCase();
  return state.requests
    .filter((rec) => matchesSearch([rec.route, rec.request?.body, rec.rejection?.reason], needle))
    .slice()
    .sort((a, b) => b.seq - a.seq);
}

// project keeps the dotted paths of the fields picker, like --fields.
function project(item, paths) {
  const out = {};
  for (const path of paths) {
    const parts = path.split(".");
    let src = parts[0] === "context" ? item.message?.context : item[parts[0]];
    for (const part of parts.slice(1)) src = src === null || src === undefined ? undefined : src[part];
    out[path] = src === undefined ? null : src;
  }
  return out;
}

// ---- rendering ---------------------------------------------------------

function clock(iso) {
  const d = new Date(iso);
  return isNaN(d) ? "-" : d.toLocaleTimeString();
}

function keyName(key) {
  return key === "" ? "(none)" : key;
}

function statusPill(item) {
  return el("span", item.statusCode, "pill " + (failed(item) ? "bad" : "ok"));
}

const DIAGNOSIS = {
  nothing_received: () => "Waiting for your app to send events to " + location.origin + ", with " +
    (writeKeyHint() || "any key") + " as the write key.",
  no_browser_traffic: () => "Server SDKs sent events but no browser request arrived. Check that the browser SDK " +
    "configUrl is " + location.origin + ", that its dataPlaneUrl is the same URL, and that the app's analytics " +
    "switch is on for local runs.",
  preflight_only: () => "The browser sent CORS preflight requests only. Open the Requests tab to see them.",
  sdk_config_rejected: () => "The browser SDK could not load its configuration from this listener.",
  sdk_loaded_no_events: () => "The SDK loaded its configuration but sent no events. Check that the app calls " +
    "track or page.",
  auth_rejected: () => "Some requests were refused because of their write key.",
  body_rejected: () => "Some requests were refused because of their body: bad JSON, batch shape, size or identity.",
  missing_write_key: (d) => d.count + " requests had no write key. RudderStack refuses these.",
  all_accepted: () => "Every request that arrived was accepted. Events that never arrived are not listed: " +
    "type their names in the Event filter to see them at 0.",
};

// writeKeyHint names the keys the listener accepts, from /info.
function writeKeyHint() {
  const info = state.info;
  if (!info) return null;
  return info.writeKeyPolicy === "allowlist" ? "one of " + info.writeKeys.join(", ") : "any key";
}

function diagnosisAction(d) {
  switch (d.code) {
    case "auth_rejected":
      return button("Show rejected requests", () => setFilters({ tab: "events", failed: true, statusCode: "401" }));
    case "body_rejected":
      return button("Show rejected requests", () => setFilters({ tab: "events", failed: true, statusCode: "4xx" }));
    case "missing_write_key":
      return button("Show requests without a write key", () => setFilters({ tab: "requests", writeKey: "" }));
    case "preflight_only":
    case "sdk_config_rejected":
    case "sdk_loaded_no_events":
      return button("Show control requests", () => setFilters({ tab: "requests", kind: "control" }));
  }
  return null;
}

function renderSummary() {
  const env = state.envelope;
  if (!env) return;
  const s = env.summary;
  $("server").textContent = location.host + " · server " + env.serverId;
  $("stat-requests").textContent = s.requests.total;
  $("stat-failed").textContent = s.requests.failed + " failed";
  $("stat-events").textContent = s.events.total;
  $("stat-types").textContent = Object.entries(s.events.byType).map(([k, v]) => k + " " + v).join(", ");
  $("stat-control").textContent = s.control.total;
  $("stat-control-sub").textContent = s.control.sourceConfig + " config, " + s.control.preflight + " preflight";
  const keys = Object.keys(s.byWriteKey || {}).sort();
  const sdks = Object.keys(s.bySource.bySdk || {});
  $("stat-sdks").textContent = sdks.length;
  $("stat-keys").textContent = sdks.join(", ") + (keys.length ? " · write keys: " + keys.map(keyName).join(", ") : "");

  $("diagnosis").replaceChildren(...s.diagnosis.map((d) => {
    const li = el("li", null, d.code === "all_accepted" ? "good" : "");
    const text = (DIAGNOSIS[d.code] || (() => d.code))(d);
    li.append(el("span", text));
    const action = diagnosisAction(d);
    if (action) {
      action.classList.add("action");
      li.append(action);
    }
    return li;
  }));

  $("by-event").replaceChildren(...Object.entries(s.byEvent).sort((a, b) => b[1] - a[1]).map(([name, n]) => {
    const tr = el("tr");
    tr.append(el("td", name), el("td", n));
    tr.addEventListener("click", () => setFilters({ tab: "events", event: name }));
    return tr;
  }));
  $("by-key").replaceChildren(...keys.map((key) => {
    const tr = el("tr");
    tr.append(el("td", keyName(key), "mono"), el("td", s.byWriteKey[key].requests), el("td", s.byWriteKey[key].events));
    tr.addEventListener("click", () => setFilters({ writeKey: key }));
    return tr;
  }));

  const select = $("f-key");
  const options = [el("option", "Any")];
  options[0].value = "";
  for (const key of keys) {
    const opt = el("option", keyName(key));
    opt.value = "k:" + key;
    options.push(opt);
  }
  select.replaceChildren(...options);
  select.value = state.writeKey === null ? "" : "k:" + state.writeKey;

  const evicted = env.evictedThrough > state.since;
  $("evicted").hidden = !evicted;
  $("evicted").textContent = evicted ? "The listener dropped its oldest requests (through #" + env.evictedThrough +
    ") to stay within its memory cap. Older rows are gone." : "";
}

function header(cells) {
  const tr = el("tr");
  for (const c of cells) tr.append(el("th", c));
  return tr;
}

function renderList() {
  const paths = listValues(state.fields);
  const isEvents = state.tab === "events";
  const rows = isEvents ? eventRows() : requestRows();
  const cols = isEvents ? ["Time", "Type", "Event", "Status", "Write key"] :
    ["#", "Time", "Method", "Route", "Status", "Outcome", "Events"];
  if (isEvents && paths.length > 0) cols.push("Fields");
  $("list-head").replaceChildren(header(cols));

  $("rows").replaceChildren(...rows.map((item) => {
    const tr = el("tr");
    tr.tabIndex = 0;
    tr.dataset.key = isEvents ? keyOf(item) : "q" + item.seq;
    if (state.selected === tr.dataset.key) tr.className = "selected";
    const status = el("td");
    status.append(statusPill(item));
    if (isEvents) {
      tr.append(el("td", clock(item.receivedAt), "mono"), el("td", item.type ?? "-"), el("td", label(item)), status,
        el("td", keyName(item.writeKey ?? ""), "mono"));
      if (paths.length > 0) tr.append(el("td", item.kind === "rejected" ? "" : JSON.stringify(project(item, paths)), "fields"));
    } else {
      tr.append(el("td", item.seq, "mono"), el("td", clock(item.receivedAt), "mono"), el("td", item.request?.method),
        el("td", item.route), status, el("td", item.outcome), el("td", item.events.length));
    }
    return tr;
  }));

  const total = isEvents ? state.events.length + state.rejected.filter((r) => r.events.length === 0).length :
    state.requests.length;
  const noun = isEvents ? "rows" : "requests";
  $("count").textContent = rows.length === total ? total + " " + noun : rows.length + " of " + total + " " + noun;
  $("empty").hidden = rows.length > 0;
  $("empty").textContent = total > 0 ? "Nothing matches the filters; " + total + " " + noun + " are hidden." :
    isEvents ? "No events yet. The listener at " + location.origin + " shows them here as your app sends them." :
      "No requests yet.";
  $("fields-foot").hidden = !(isEvents && paths.length > 0);
}

function pretty(v) {
  return JSON.stringify(v ?? null, null, 2);
}

function compactOf(ev) {
  const { message, enrichedMessage, ...rest } = ev;
  return rest;
}

function selectedItem() {
  if (!state.selected) return null;
  if (state.selected.startsWith("q")) {
    const seq = Number(state.selected.slice(1));
    return state.requests.find((r) => r.seq === seq) || state.requestCache.get(seq) || null;
  }
  return eventRows().find((ev) => keyOf(ev) === state.selected) || null;
}

function facts(pairs) {
  const nodes = [];
  for (const [k, v] of pairs) {
    if (v === null || v === undefined || v === "") continue;
    nodes.push(el("dt", k), el("dd", v, "mono"));
  }
  $("detail-facts").replaceChildren(...nodes);
}

function subtabs(names) {
  $("detail-tabs").replaceChildren(...names.map(([id, text]) => {
    const b = button(text, () => {
      state.detailTab = id;
      renderDetail();
    }, "btn" + (state.detailTab === id ? " on" : ""));
    return b;
  }));
}

async function loadRequest(seq) {
  if (!state.requestCache.has(seq)) {
    state.requestCache.set(seq, await getJSON("requests/" + seq + "?view=full&maxBytes=0"));
  }
  return state.requestCache.get(seq);
}

async function renderDetail() {
  const item = selectedItem();
  $("detail-empty").hidden = Boolean(item);
  $("detail-body").hidden = !item;
  if (!item) return;
  const pill = $("detail-status");
  pill.textContent = item.statusCode;
  pill.className = "pill " + (failed(item) ? "bad" : "ok");
  const related = [];

  if (state.selected.startsWith("q") || item.kind === "rejected") {
    const rec = item.kind === "rejected" ? await loadRequest(item.seq) : item;
    $("detail-title").textContent = "Request #" + rec.seq + " " + rec.route;
    facts([["received", rec.receivedAt], ["kind", rec.kind], ["method", rec.request?.method],
      ["outcome", rec.outcome], ["rejection", rec.rejection && rec.rejection.stage + ": " + rec.rejection.reason],
      ["write key", keyName(rec.writeKey ?? "")]]);
    const tabs = [["body", "Raw body"], ["headers", "Headers"], ["response", "Response"], ["full", "Full"]];
    if (!tabs.some(([id]) => id === state.detailTab)) state.detailTab = "body";
    subtabs(tabs);
    const views = { body: rec.request?.body, headers: rec.request?.headers, response: rec.response, full: rec };
    const v = views[state.detailTab];
    $("detail-label").textContent = tabs.find(([id]) => id === state.detailTab)[1];
    $("detail-json").textContent = typeof v === "string" ? prettyString(v) : pretty(v);
    for (const ev of rec.events || []) {
      related.push(button("Event " + (ev.event ?? ev.type ?? ev.idx), () => {
        setFilters({ tab: "events" });
        select(rec.seq + "." + ev.idx);
      }));
    }
  } else {
    $("detail-title").textContent = label(item);
    facts([["seq", seqLabel(item)], ["type", item.type], ["received", item.receivedAt], ["route", item.route],
      ["write key", keyName(item.writeKey ?? "")], ["userId", item.userId], ["anonymousId", item.anonymousId],
      ["messageId", item.messageId]]);
    const tabs = [["compact", "Compact"], ["full", "Full"], ["request", "Request"]];
    if (!tabs.some(([id]) => id === state.detailTab)) state.detailTab = "compact";
    subtabs(tabs);
    $("detail-label").textContent = tabs.find(([id]) => id === state.detailTab)[1];
    if (state.detailTab === "request") {
      $("detail-json").textContent = "Loading request #" + item.seq + "...";
      try {
        const rec = await loadRequest(item.seq);
        if (state.selected === keyOf(item)) $("detail-json").textContent = pretty(rec);
      } catch (err) {
        $("detail-json").textContent = "Could not load request #" + item.seq + ": " + err.message;
      }
    } else {
      $("detail-json").textContent = pretty(state.detailTab === "full" ? item : compactOf(item));
    }
    related.push(button("Open request #" + item.seq, () => {
      setFilters({ tab: "requests" });
      select("q" + item.seq);
    }));
  }
  $("related").replaceChildren(...related);
}

function prettyString(s) {
  try {
    return pretty(JSON.parse(s));
  } catch {
    return s;
  }
}

function render() {
  for (const node of document.querySelectorAll("[data-tab]")) node.hidden = node.dataset.tab !== state.tab;
  $("tab-events").setAttribute("aria-selected", String(state.tab === "events"));
  $("tab-requests").setAttribute("aria-selected", String(state.tab === "requests"));
  $("pause").textContent = state.paused ? "Resume" : "Pause";
  $("pause").classList.toggle("on", state.paused);
  $("only-new").textContent = state.since > 0 ? "Show all" : "Show only new";
  $("live-dot").className = "dot " + (state.paused ? "" : "on");
  renderSummary();
  renderList();
}

// ---- loading ------------------------------------------------------------

function restart() {
  state.generation++;
  if (state.controller) state.controller.abort();
  state.cursor = state.since;
  state.events = [];
  state.rejected = [];
  state.requests = [];
  state.requestCache.clear();
  loop(state.generation);
}

async function loadOnce(gen, wait) {
  const ctl = new AbortController();
  state.controller = ctl;
  let more = true;
  while (more && gen === state.generation) {
    const page = await getJSON(eventsQuery(state.cursor, wait), ctl.signal);
    if (gen !== state.generation) return;
    state.serverId = page.serverId;
    state.envelope = page;
    state.events.push(...page.events);
    state.cursor = page.cursor;
    more = page.hasMore;
    wait = null;
  }
  if (state.events.length > MAX_EVENTS) state.events.splice(0, state.events.length - MAX_EVENTS);
  const [rejected, requests] = await Promise.all([
    getJSON(requestsQuery("all", true), ctl.signal),
    getJSON(requestsQuery(state.kind, state.failed), ctl.signal),
  ]);
  if (gen !== state.generation) return;
  state.rejected = rejected.requests;
  state.requests = requests.requests;
  state.requestsEnvelope = requests;
}

async function loop(gen) {
  let first = true;
  while (gen === state.generation) {
    try {
      await loadOnce(gen, first || state.paused ? null : LIVE_WAIT);
      first = false;
      render();
      if (state.selected && !selectedItem()) {
        state.selected = null;
        renderDetail();
      }
    } catch (err) {
      if (gen !== state.generation) return;
      if (err.code === "server_changed") {
        state.serverId = null;
        restart();
        return;
      }
      $("live-dot").className = "dot off";
      $("live-dot").title = "Listener unreachable: " + err.message;
      await sleep(2000);
    }
    while (state.paused && gen === state.generation) await sleep(300);
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// ---- controls -----------------------------------------------------------

function setFilters(change) {
  const reload = Object.keys(change).some((k) => !["tab", "q", "fields", "paused"].includes(k));
  Object.assign(state, change);
  syncControls();
  writeURL();
  render();
  if (state.selected && !selectedItem()) state.selected = null;
  renderDetail();
  if (reload) restart();
}

function syncControls() {
  $("search").value = state.q;
  $("f-event").value = state.event;
  $("f-type").value = state.type;
  $("f-user").value = state.userId;
  $("f-anon").value = state.anonymousId;
  $("f-status").value = state.statusCode;
  $("f-kind").value = state.kind;
  $("f-failed").checked = state.failed;
  $("f-fields").value = state.fields;
}

function select(key) {
  state.selected = key;
  renderList();
  renderDetail();
}

function onChange(id, field, read) {
  $(id).addEventListener("change", (e) => setFilters({ [field]: read(e.target) }));
}

onChange("f-event", "event", (t) => t.value.trim());
onChange("f-type", "type", (t) => t.value);
onChange("f-user", "userId", (t) => t.value.trim());
onChange("f-anon", "anonymousId", (t) => t.value.trim());
onChange("f-status", "statusCode", (t) => t.value.trim());
onChange("f-kind", "kind", (t) => t.value);
onChange("f-failed", "failed", (t) => t.checked);
onChange("f-key", "writeKey", (t) => (t.value === "" ? null : t.value.slice(2)));
$("f-fields").addEventListener("input", (e) => setFilters({ fields: e.target.value }));
$("search").addEventListener("input", (e) => setFilters({ q: e.target.value }));
$("tab-events").addEventListener("click", () => setFilters({ tab: "events" }));
$("tab-requests").addEventListener("click", () => setFilters({ tab: "requests" }));
$("pause").addEventListener("click", () => {
  setFilters({ paused: !state.paused });
  if (!state.paused) restart();
});
$("only-new").addEventListener("click", () => setFilters({ since: state.since > 0 ? 0 : state.cursor }));

$("rows").addEventListener("click", (e) => {
  const tr = e.target.closest("tr");
  if (tr && tr.dataset.key) select(tr.dataset.key);
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

async function copyText(btn, text) {
  const was = btn.textContent;
  try {
    await navigator.clipboard.writeText(text);
    btn.textContent = "Copied";
  } catch {
    btn.textContent = "Copy failed";
  }
  setTimeout(() => { btn.textContent = was; }, 1200);
}

$("copy-detail").addEventListener("click", (e) => copyText(e.target, $("detail-json").textContent));
$("copy-fields").addEventListener("click", (e) => {
  const paths = listValues(state.fields);
  const rows = eventRows().filter((ev) => ev.kind !== "rejected");
  copyText(e.target, pretty(rows.map((ev) => ({ seq: ev.seq, idx: ev.idx, ...project(ev, paths) }))));
});

// Export saves the envelope the API returned, with the list as filtered on
// this page.
$("export").addEventListener("click", () => {
  const isEvents = state.tab === "events";
  const base = isEvents ? state.envelope : state.requestsEnvelope;
  const doc = isEvents ?
    { ...base, events: eventRows().filter((ev) => ev.kind !== "rejected") } :
    { ...base, requests: requestRows() };
  const blob = new Blob([JSON.stringify(doc, null, 2)], { type: "application/json" });
  const a = el("a");
  a.href = URL.createObjectURL(blob);
  a.download = (isEvents ? "events" : "requests") + "-" + (base?.serverId || "capture") + ".json";
  document.body.append(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000);
});

readURL();
syncControls();
render();
getJSON("info").then((info) => {
  state.info = info;
  renderSummary();
}).catch(() => {});
restart();
