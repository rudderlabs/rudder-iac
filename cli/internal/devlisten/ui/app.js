// The review page of the local listener. It reads the capture through the
// query API under ../v1/ only, and writes every captured value as text.
'use strict';

(() => {
  const API = '../v1/';
  const POLL_WAIT = '5s';
  const RETRY_MS = 5000;
  // Every list loads whole, so the page size changes only the number of
  // round trips. The largest page keeps that number small.
  const PAGE_LIMIT = '1000';

  // Filter keys as they appear in the page URL. Lists repeat the key, as
  // the API does. serverId and the pause state never come from the URL, so
  // a link from another site cannot set them.
  const LISTS = ['event', 'type', 'writeKey', 'fields', 'statusCode', 'rfields'];
  const DEFAULTS = {
    q: '', event: [], type: [], userId: '', anonymousId: '', writeKey: [], since: '',
    view: '', fields: [], kind: 'all', statusCode: [], failed: false, messageId: '', rview: '', rfields: [],
  };
  // LABELS names a query parameter by the field that sets it.
  const LABELS = {
    since: 'Received', event: 'Event name', type: 'Type', writeKey: 'Write keys', userId: 'User ID',
    anonymousId: 'Anonymous ID', view: 'Show', fields: 'Only these fields', kind: 'Kind',
    statusCode: 'Status', failed: 'Failed only', messageId: 'Message ID',
  };
  const VIEWS = {
    events: [['', 'Whole event'], ['compact', 'Without automatic context']],
    requests: [['', 'Main keys'], ['full', 'Whole record'], ['list', 'Minimal']],
  };

  const state = {
    serverId: '',
    url: '',
    tab: 'events',
    f: { ...DEFAULTS },
    paused: false,
    stopped: '',
    polling: false,
    poll: null,
    summary: null,
    cursor: 0,
    evictedThrough: 0,
    generation: 0,
    nextKey: 0,
    lists: { events: newList(), requests: newList() },
    selected: null,
    detailToken: 0,
    requestDetailView: 'compact',
  };

  // A list sets its cursor only when a whole load ends. Its loads run one
  // after another on queue, so a live update never reads pages that a
  // reload is still reading.
  function newList() {
    return { rows: [], cursor: null, queue: Promise.resolve() };
  }

  const $ = (id) => document.getElementById(id);
  const form = () => $('filters');

  function el(tag, props, ...kids) {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(props || {})) {
      if (value == null || value === false) continue;
      if (key === 'class') node.className = value;
      else if (key === 'text') node.textContent = value;
      else if (key.startsWith('on')) node.addEventListener(key.slice(2), value);
      else node.setAttribute(key, value === true ? '' : String(value));
    }
    for (const kid of kids.flat()) {
      if (kid != null && kid !== false) node.append(kid);
    }
    return node;
  }

  // fill drops the null and false of optional parts, which replaceChildren
  // would write as text.
  function fill(node, ...kids) {
    node.replaceChildren(...kids.flat().filter((kid) => kid != null && kid !== false));
  }

  const str = (v) => (v == null ? '' : typeof v === 'string' ? v : JSON.stringify(v));
  const plural = (n, one, many) => `${n} ${n === 1 ? one : many || one + 's'}`;
  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

  // ---- The page URL holds the filters, so a view can be shared.

  function readURL() {
    const p = new URLSearchParams(location.search);
    state.tab = p.get('tab') === 'requests' ? 'requests' : 'events';
    for (const key of Object.keys(DEFAULTS)) {
      if (!p.has(key)) continue;
      if (LISTS.includes(key)) state.f[key] = p.getAll(key);
      else if (key === 'failed') state.f.failed = p.get(key) === 'true';
      else state.f[key] = p.get(key);
    }
  }

  function writeURL() {
    const p = new URLSearchParams();
    if (state.tab !== 'events') p.set('tab', state.tab);
    for (const [key, value] of Object.entries(state.f)) {
      if (LISTS.includes(key)) value.forEach((v) => p.append(key, v));
      else if (key === 'failed') { if (value) p.set(key, 'true'); }
      else if (value !== DEFAULTS[key]) p.set(key, value);
    }
    const query = p.toString();
    history.replaceState(null, '', query ? '?' + query : location.pathname);
  }

  // ---- Form fields to filters and back. Several values are comma separated
  // and \, is a comma inside a value; "(none)" stands for an empty write key.
  // A field the person did not edit keeps the values it shows, so a value
  // set by a link or a click stays literal: spaces, commas and "(none)".

  const splitList = (text) => text.split(/(?<!\\),/).map((s) => s.trim().replace(/\\,/g, ',')).filter((s) => s !== '');
  const joinList = (values) => values.map((v) => v.replace(/,/g, '\\,')).join(', ');
  const keysIn = (text) => splitList(text).map((s) => (s === '(none)' ? '' : s));
  const keysOut = (keys) => joinList(keys.map((k) => (k === '' ? '(none)' : k)));
  const formText = {};

  function showList(name, text) {
    form().elements[name].value = text;
    formText[name] = text;
  }

  function readList(name, current, parse) {
    const text = form().elements[name].value;
    return text === formText[name] ? current : (parse || splitList)(text);
  }

  function fillForm() {
    const f = form().elements;
    const ev = state.tab === 'events';
    f.q.value = state.f.q;
    showList('event', joinList(state.f.event));
    showList('writeKey', keysOut(state.f.writeKey));
    f.userId.value = state.f.userId;
    f.anonymousId.value = state.f.anonymousId;
    f.kind.value = state.f.kind;
    showList('statusCode', joinList(state.f.statusCode));
    f.failed.checked = state.f.failed;
    f.messageId.value = state.f.messageId;
    form().querySelectorAll('input[name="type"]').forEach((box) => { box.checked = state.f.type.includes(box.value); });
    showSince();

    const view = f.view;
    view.replaceChildren(...VIEWS[state.tab].map(([value, text]) => el('option', { value, text })));
    view.value = ev ? state.f.view : state.f.rview;
    showList('fields', joinList(ev ? state.f.fields : state.f.rfields));
    f.fields.placeholder = ev ? 'properties, context.page.path' : 'rejection, request.headers';

    form().querySelectorAll('.for-events').forEach((n) => { n.hidden = !ev; });
    form().querySelectorAll('.for-requests').forEach((n) => { n.hidden = ev; });
  }

  // A since value that no option names, such as a time or a request number,
  // shows in the custom field.
  function showSince() {
    const f = form().elements;
    const preset = [...f.since.options].some((o) => o.value === state.f.since && o.value !== 'custom');
    f.since.value = preset ? state.f.since : 'custom';
    f.sinceAt.value = preset ? '' : state.f.since;
    showCustomSince(!preset);
  }

  function showCustomSince(on) {
    $('since-custom').hidden = !on;
    $('since-hint').hidden = !on;
  }

  function readForm() {
    const f = form().elements;
    const ev = state.tab === 'events';
    state.f.event = readList('event', state.f.event);
    state.f.writeKey = readList('writeKey', state.f.writeKey, keysIn);
    state.f.userId = f.userId.value.trim();
    state.f.anonymousId = f.anonymousId.value.trim();
    state.f.type = [...form().querySelectorAll('input[name="type"]:checked')].map((b) => b.value);
    state.f.kind = f.kind.value;
    state.f.statusCode = readList('statusCode', state.f.statusCode);
    state.f.failed = f.failed.checked;
    state.f.messageId = f.messageId.value.trim();
    state.f.since = f.since.value === 'custom' ? f.sinceAt.value.trim() : f.since.value;
    state.f[ev ? 'view' : 'rview'] = f.view.value;
    state.f[ev ? 'fields' : 'rfields'] = readList('fields', ev ? state.f.fields : state.f.rfields);
  }

  // ---- API calls. Each filter is the query parameter of the same name.

  class ApiError extends Error {
    constructor(status, body) {
      const e = (body && body.error) || {};
      super(e.message || `The listener answered ${status}.`);
      this.status = status;
      this.code = e.code || '';
      this.param = e.param || '';
    }
  }

  async function api(path, params, signal) {
    const query = params ? params.toString() : '';
    let res;
    try {
      res = await fetch(API + path + (query ? '?' + query : ''), { signal, cache: 'no-store' });
    } catch (err) {
      if (signal && signal.aborted) throw err;
      throw new ApiError(0, { error: { code: 'unreachable', message: 'The listener cannot be reached.' } });
    }
    if (!res.ok) {
      let body = null;
      try { body = await res.json(); } catch { /* not JSON */ }
      throw new ApiError(res.status, body);
    }
    return res;
  }

  function eventsQuery(since) {
    const f = state.f;
    const p = new URLSearchParams({ serverId: state.serverId });
    if (since !== '') p.set('since', since);
    f.event.forEach((v) => p.append('event', v));
    f.type.forEach((v) => p.append('type', v));
    f.writeKey.forEach((v) => p.append('writeKey', v));
    if (f.userId) p.set('userId', f.userId);
    if (f.anonymousId) p.set('anonymousId', f.anonymousId);
    if (f.fields.length) f.fields.forEach((v) => p.append('fields', v));
    else if (f.view) p.set('view', f.view);
    p.set('limit', PAGE_LIMIT);
    return p;
  }

  function requestsQuery(since, view) {
    const f = state.f;
    const p = new URLSearchParams({ serverId: state.serverId, kind: f.kind, maxBytes: '0' });
    if (since !== '') p.set('since', since);
    f.statusCode.forEach((v) => p.append('statusCode', v));
    f.writeKey.forEach((v) => p.append('writeKey', v));
    if (f.failed) p.set('failed', 'true');
    if (f.messageId) p.set('messageId', f.messageId);
    if (view) p.set('view', view);
    else if (f.rfields.length) f.rfields.forEach((v) => p.append('fields', v));
    else p.set('view', f.rview || 'compact');
    p.set('limit', PAGE_LIMIT);
    return p;
  }

  // parseExact keeps the digits of a number that a JavaScript number would
  // change, such as 9007199254740993, so the detail and the export show it
  // as sent. A browser without JSON.rawJSON parses as usual.
  function parseExact(text) {
    if (typeof JSON.rawJSON !== 'function') return JSON.parse(text);
    return JSON.parse(text, (key, value, context) => (typeof value === 'number' && context &&
      typeof context.source === 'string' && String(value) !== context.source ? JSON.rawJSON(context.source) : value));
  }

  async function fetchRecord(seq, view) {
    const p = new URLSearchParams({ serverId: state.serverId, view, maxBytes: '0' });
    return parseExact(await (await api('requests/' + seq, p)).text());
  }

  async function fetchEvents(since) {
    const res = await api('events', eventsQuery(since));
    const text = await res.text();
    const rows = text.split('\n').filter((line) => line !== '').map((line) => {
      let obj = null;
      try { obj = parseExact(line); } catch { /* shown as text */ }
      return { key: ++state.nextKey, line, obj, search: line.toLowerCase() };
    });
    return {
      rows,
      cursor: Number(res.headers.get('X-Local-Cursor')),
      hasMore: res.headers.get('X-Local-Has-More') === 'true',
    };
  }

  async function fetchRequests(since, view) {
    const env = parseExact(await (await api('requests', requestsQuery(since, view))).text());
    const rows = env.requests.map((obj) => ({ key: obj.seq, obj, search: JSON.stringify(obj).toLowerCase() }));
    return { rows, cursor: env.cursor, hasMore: env.hasMore, evictedThrough: env.evictedThrough };
  }

  // loadList queues a load of the named list. A load with no since reads
  // after the cursor of the load before it; it does nothing when that load
  // failed.
  function loadList(name, gen, since, fresh) {
    const list = state.lists[name];
    const run = list.queue.then(() => {
      if (gen !== state.generation) return undefined;
      if (since == null && list.cursor == null) return undefined;
      return loadPages(name, list, gen, since == null ? String(list.cursor) : since, fresh);
    });
    list.queue = run.catch(() => {});
    return run;
  }

  // loadPages follows the cursor until the listener has nothing more, so a
  // list of more than one page loads whole.
  async function loadPages(name, list, gen, since, fresh) {
    let next = since;
    for (;;) {
      const page = name === 'events' ? await fetchEvents(next) : await fetchRequests(next);
      if (gen !== state.generation) return;
      list.rows.push(...page.rows);
      if (name === state.tab) addRows(page.rows, fresh);
      if (!page.hasMore) {
        list.cursor = page.cursor;
        return;
      }
      next = String(page.cursor);
    }
  }

  async function refreshSummary() {
    const s = await (await api('events', new URLSearchParams({ view: 'counts', serverId: state.serverId }))).json();
    state.summary = s;
    state.cursor = s.cursor;
    renderSummary();
  }

  // ---- Loading and live updates.

  async function reload() {
    const gen = ++state.generation;
    state.lists = { events: newList(), requests: newList() };
    state.stopped = '';
    clearDetail();
    showFilterError('');
    renderHead();
    $('list-body').replaceChildren();
    updateCount();
    try {
      await refreshSummary();
      await Promise.all(['events', 'requests'].map((name) => loadList(name, gen, state.f.since, false)));
    } catch (err) {
      if (gen === state.generation && handleError(err)) {
        await sleep(RETRY_MS);
        if (gen === state.generation) reload();
      }
      return;
    }
    if (gen !== state.generation) return;
    updateCount();
    pollLoop();
  }

  async function refresh() {
    const gen = state.generation;
    await refreshSummary();
    await Promise.all(['events', 'requests'].map((name) => loadList(name, gen, null, true)));
    if (gen === state.generation) updateCount();
  }

  // pollLoop long-polls the unfiltered counts. The listener answers at once
  // when an event arrives and after the wait otherwise, so refused requests
  // show within one wait.
  async function pollLoop() {
    if (state.polling) return;
    state.polling = true;
    while (!state.paused && !state.stopped) {
      const ctrl = new AbortController();
      state.poll = ctrl;
      try {
        const params = new URLSearchParams({
          view: 'counts', since: String(state.cursor), wait: POLL_WAIT, serverId: state.serverId,
        });
        const s = await (await api('events', params, ctrl.signal)).json();
        setLive('live');
        if (s.cursor !== state.cursor) await refresh();
      } catch (err) {
        if (ctrl.signal.aborted) break;
        if (!handleError(err)) break;
        await sleep(RETRY_MS);
      }
    }
    state.poll = null;
    state.polling = false;
  }

  // handleError shows the error as text and reports whether a retry may
  // help. No retry runs after a 4xx: the same request gets the same answer.
  function handleError(err) {
    if (!(err instanceof ApiError)) {
      showNotice('fatal', 'bad', `The page failed: ${err.message}`);
      state.stopped = 'error';
      setLive('down');
      return false;
    }
    if (err.status === 0 || (err.status >= 500 && err.code !== 'shutting_down')) {
      showNotice('fatal', 'bad', 'The listener cannot be reached. It may have stopped. The page tries again every 5 seconds.');
      setLive('down');
      return true;
    }
    state.stopped = err.code || String(err.status);
    setLive('down');
    updateCount();
    switch (err.code) {
      case 'invalid_parameter':
      case 'unknown_parameter':
        showFilterError(`The ${LABELS[err.param] || err.param || 'filter'} filter is not valid. ` +
          `${err.message.replace(/[^.]$/, '$&.')} Change it to continue.`);
        break;
      case 'server_changed':
        showNotice('fatal', 'bad', 'The listener restarted, so this page shows a capture that is gone.',
          { label: 'Start over', run: () => location.reload() });
        break;
      case 'shutting_down':
        showNotice('fatal', 'bad', 'The listener is stopping. Updates stopped.',
          { label: 'Try again', run: () => location.reload() });
        break;
      default:
        showNotice('fatal', 'bad', `The listener refused the page: ${err.message}`);
    }
    return false;
  }

  function setLive(mode) {
    const dot = $('live-dot');
    dot.className = 'dot ' + mode;
    if (mode === 'live') showNotice('fatal');
    $('live-text').textContent = { live: 'Live', paused: 'Paused', down: 'Not updating' }[mode];
  }

  function togglePause() {
    state.paused = !state.paused;
    const button = $('pause');
    button.setAttribute('aria-pressed', String(state.paused));
    button.textContent = state.paused ? 'Resume' : 'Pause';
    if (state.paused) {
      if (state.poll) state.poll.abort();
      setLive('paused');
      return;
    }
    if (state.stopped) return;
    setLive('live');
    refresh().then(pollLoop, (err) => { if (handleError(err)) pollLoop(); });
  }

  // ---- Whole capture: tiles, notices and counts.

  function tile(title, num, sub, bad) {
    return el('div', { class: 'tile' + (bad ? ' bad' : '') },
      el('h2', { text: title }), el('div', { class: 'num', text: String(num) }), el('div', { class: 'sub', text: sub }));
  }

  const top = (counts, max) => Object.entries(counts || {}).sort((a, b) => b[1] - a[1]).slice(0, max || 4);
  const listed = (counts) => top(counts).map(([k, n]) => `${k === '' ? '(none)' : k} ${n}`).join(' · ');

  function renderSummary() {
    const s = state.summary.summary;
    const sdks = Object.keys(s.bySource.bySdk);
    const keys = Object.keys(s.byWriteKey);
    $('tiles').replaceChildren(
      tile('Requests', s.requests.total, `${s.requests.failed} refused`, s.requests.failed > 0),
      tile('Events', s.events.total, listed(s.events.byType) || 'none yet'),
      tile('Rejected events', s.rejected.events, listed(s.rejected.byEvent) || 'none', s.rejected.events > 0),
      tile('Control requests', s.control.total, `${s.control.sourceConfig} settings · ${s.control.preflight} preflight`),
      tile('SDKs', sdks.length, sdks.join(', ') || 'none yet'),
      tile('Write keys', keys.length, keys.map((k) => (k === '' ? '(none)' : k)).join(', ') || 'none yet'),
    );

    const byEvent = $('by-event');
    byEvent.replaceChildren(...top(s.byEvent, 500).map(([name, n]) => el('li', null,
      el('button', { type: 'button', text: name, onclick: () => setFilters('events', { event: [name] }) }),
      el('span', { class: 'n', text: String(n) }))));
    if (!byEvent.childElementCount) byEvent.append(el('li', { text: 'No events yet.' }));
    const rejected = $('by-rejected');
    rejected.replaceChildren(...top(s.rejected.byEvent, 500).map(([name, n]) => el('li', null,
      el('span', { text: name }), el('span', { class: 'n', text: String(n) }))));
    if (!rejected.childElementCount) rejected.append(el('li', { text: 'None.' }));
    $('by-key').replaceChildren(...keys.map((k) => el('li', null,
      el('span', { class: 'mono', text: k === '' ? '(none)' : k }),
      el('span', { class: 'n', text: `${plural(s.byWriteKey[k].requests, 'request')}, ${plural(s.byWriteKey[k].events, 'event')}` }))));

    renderDiagnosis(s.diagnosis);
    const evicted = state.summary.evictedThrough;
    if (evicted > 0) {
      dropEvicted(evicted);
      showNotice('evicted', 'warn', `The listener dropped its oldest ${plural(evicted, 'request')} ` +
        'to stay within its memory limit. The requests list no longer shows them. ' +
        'The events list keeps the events it already shows until the page reloads.');
    }
    updateCount();
  }

  // dropEvicted removes the request rows the listener no longer holds, so
  // the list and its export agree with the listener.
  function dropEvicted(through) {
    const list = state.lists.requests;
    if (!list.rows.some((r) => r.obj.seq <= through)) return;
    const sel = state.selected;
    for (const row of list.rows) {
      if (row.obj.seq > through) continue;
      if (row.node) row.node.remove();
      if (sel && sel.tab === 'requests' && sel.key === row.key) clearDetail();
    }
    list.rows = list.rows.filter((r) => r.obj.seq > through);
    if (state.tab === 'requests') keepRowInTabOrder();
  }

  // Each diagnosis shows in the page's own words, with an action on this page.
  const ACTIONS = {
    rejected: { label: 'Show refused requests', run: () => setFilters('requests', { failed: true }) },
    auth: { label: 'Show refused requests', run: () => setFilters('requests', { failed: true, statusCode: ['401'] }) },
    control: { label: 'Show control requests', run: () => setFilters('requests', { kind: 'control' }) },
    controlFailed: { label: 'Show refused settings requests', run: () => setFilters('requests', { kind: 'control', failed: true }) },
    noKey: { label: 'Show them', run: () => setFilters('requests', { writeKey: [''] }) },
    counts: { label: 'Show counts by event', run: () => { $('breakdown').open = true; $('breakdown').querySelector('summary').focus(); } },
  };

  function diagnosisText(d) {
    const n = d.count;
    const requests = plural(n, 'request was', 'requests were');
    switch (d.code) {
      case 'nothing_received':
        return ['info', `Nothing has arrived yet. Point your app's RudderStack SDK at ${state.url}. Events show here as they arrive.`];
      case 'nothing_new':
        return ['info', 'No new request arrived in this time window.'];
      case 'preflight_only':
        return ['warn', `The browser sent ${plural(n, 'preflight check')} but no request with events. A preflight check carries no data.`, 'control'];
      case 'sdk_config_rejected':
        return ['warn', 'The SDK asked for its settings and was refused: its write key is not on the listener\'s list of allowed keys.', 'controlFailed'];
      case 'sdk_loaded_no_events':
        return ['warn', 'The SDK loaded its settings but sent no event. Check that your tracking code runs, and that no plugin or consent setting stops it.', 'control'];
      case 'no_browser_traffic':
        return ['info', 'No browser traffic arrived. If your app also runs the browser SDK, the browser does not reach this listener.'];
      case 'auth_rejected':
        return ['warn', `${requests} refused because the write key is missing or not allowed.`, 'auth'];
      case 'body_rejected':
        return ['warn', `${requests} refused while the body was read: bad gzip, an empty body, invalid JSON, a wrong batch shape, no user or anonymous ID, or too large.`, 'rejected'];
      case 'missing_write_key':
        return ['warn', `${plural(n, 'request')} arrived without a write key. This listener accepts them; RudderStack refuses them.`, 'noKey'];
      case 'all_accepted':
        return ['ok', `Every request was accepted. An event that never arrived does not show here, so check the counts by event.`, 'counts'];
      case 'filtered_empty':
        return ['info', 'Requests arrived, but the filters match none. Names are exact and case-sensitive.'];
    }
    return ['info', d.message];
  }

  function renderDiagnosis(items) {
    $('notices').querySelectorAll('[data-diagnosis]').forEach((n) => n.remove());
    for (const d of items) {
      const [level, text, action] = diagnosisText(d);
      const node = notice(level, text, ACTIONS[action]);
      node.dataset.diagnosis = d.code;
      $('notices').append(node);
    }
  }

  function notice(level, text, action) {
    return el('div', { class: 'notice ' + level },
      el('p', { text }),
      action && el('button', { type: 'button', text: action.label, onclick: action.run }));
  }

  // showNotice keeps one notice per id at the top; no text removes it.
  function showNotice(id, level, text, action) {
    const old = $('notices').querySelector(`[data-id="${id}"]`);
    if (!text) { if (old) old.remove(); return; }
    const node = notice(level, text, action);
    node.dataset.id = id;
    if (old) old.replaceWith(node);
    else $('notices').prepend(node);
  }

  function showFilterError(text) {
    const p = $('filter-error');
    p.textContent = text;
    p.hidden = !text;
  }

  // ---- The list.

  // timeOf takes strings only: an SDK can send any JSON as a timestamp.
  const timeOf = (o) => (o && [o.originalTimestamp, o.timestamp, o.sentAt].find((v) => typeof v === 'string' && v !== '')) || '';

  function clock(iso) {
    if (typeof iso !== 'string') return str(iso);
    const d = new Date(iso);
    if (!iso || Number.isNaN(d.getTime())) return iso;
    return d.toLocaleTimeString([], { hour12: false });
  }

  function eventLabel(o) {
    if (!o || typeof o !== 'object') return '';
    switch (o.type) {
      case 'page':
      case 'screen':
        return str(o.name ?? (o.properties && o.properties.name) ?? o.event);
      case 'identify':
        return str(o.userId ?? o.anonymousId);
      case 'group':
        return str(o.groupId);
      case 'alias':
        return `${str(o.previousId)} → ${str(o.userId)}`;
    }
    return str(o.event);
  }

  function pick(o, path) {
    let values = [o];
    for (const key of path.split('.')) {
      values = values.flatMap((v) => (Array.isArray(v) ? v : [v]))
        .filter((v) => v && typeof v === 'object' && key in v).map((v) => v[key]);
    }
    return values.length === 1 ? values[0] : values.length ? values : undefined;
  }

  function columns() {
    const f = state.f;
    if (state.tab === 'events') {
      if (f.fields.length) return f.fields.map((path) => ({ title: path, cell: (r) => str(pick(r.obj, path)), mono: true }));
      const cols = [
        { title: 'Sent (SDK clock)', cell: (r) => clock(timeOf(r.obj)), tip: (r) => timeOf(r.obj), mono: true },
        { title: 'Type', cell: (r) => str(r.obj && r.obj.type) },
        { title: 'Name', cell: (r) => (r.obj ? eventLabel(r.obj) : r.line) },
        { title: 'User', cell: (r) => (r.obj ? str(r.obj.userId) || (r.obj.anonymousId ? 'anon ' + str(r.obj.anonymousId) : '') : '') },
      ];
      if (f.view === 'compact') cols.push({ title: 'Properties', cell: (r) => str(r.obj && r.obj.properties), mono: true });
      return cols;
    }
    if (f.rfields.length) {
      return [{ title: '#', cell: (r) => str(r.obj.seq), mono: true },
        ...f.rfields.map((path) => ({ title: path, cell: (r) => str(pick(r.obj, path)), mono: true }))];
    }
    return [
      { title: '#', cell: (r) => str(r.obj.seq), mono: true },
      { title: 'Received', cell: (r) => clock(r.obj.receivedAt), tip: (r) => str(r.obj.receivedAt), mono: true },
      { title: 'Kind', cell: (r) => str(r.obj.kind) },
      { title: 'Request', cell: (r) => `${str(r.obj.method || (r.obj.request && r.obj.request.method))} ${str(r.obj.route)}` },
      { title: 'Status', badge: (r) => r.obj.statusCode },
      { title: 'Events', cell: (r) => (Array.isArray(r.obj.events) ? r.obj.events.map((e) => str(e.event || e.type) || '?').join(', ') : str(r.obj.eventCount)) },
      { title: 'Reason', cell: (r) => (r.obj.rejection ? r.obj.rejection.reason : '') },
    ];
  }

  function statusBadge(code) {
    const level = code >= 200 && code < 300 ? 'ok' : code >= 400 ? 'bad' : 'info';
    return el('span', { class: 'badge ' + level, text: str(code) });
  }

  function renderHead() {
    $('list-head').replaceChildren(...columns().map((c) => el('th', { scope: 'col', text: c.title })));
    $('list-caption').textContent = state.tab === 'events' ? 'Accepted events, newest first' : 'Requests, newest first';
  }

  function rowNode(row, cols) {
    const tr = el('tr', { tabindex: '-1', 'aria-selected': 'false', 'data-key': row.key, onclick: () => select(row.key, true) });
    for (const c of cols) {
      if (c.badge) {
        tr.append(el('td', null, statusBadge(c.badge(row))));
        continue;
      }
      const text = c.cell(row);
      tr.append(el('td', { class: c.mono ? 'mono' : null, title: c.tip ? c.tip(row) : text.length > 40 ? text.slice(0, 500) : null, text }));
    }
    row.node = tr;
    tr.hidden = !matchesSearch(row);
    return tr;
  }

  // addRows puts the newest rows on top.
  function addRows(rows, fresh) {
    const body = $('list-body');
    const cols = columns();
    for (const row of rows) {
      const tr = rowNode(row, cols);
      if (fresh) {
        tr.classList.add('fresh');
        setTimeout(() => tr.classList.remove('fresh'), 4000);
      }
      body.prepend(tr);
    }
    keepRowInTabOrder();
  }

  // keepRowInTabOrder keeps one visible row in the tab order, so a keyboard
  // reaches the list after a search hides the row that held it.
  function keepRowInTabOrder() {
    const body = $('list-body');
    const current = body.querySelector('tr[tabindex="0"]');
    if (current && !current.hidden) return;
    if (current) current.tabIndex = -1;
    const first = body.querySelector('tr:not([hidden])');
    if (first) first.tabIndex = 0;
  }

  function renderRows() {
    renderHead();
    const body = $('list-body');
    body.replaceChildren();
    addRows(state.lists[state.tab].rows, false);
    updateCount();
  }

  const matchesSearch = (row) => !state.f.q || row.search.includes(state.f.q.toLowerCase());

  function applySearch() {
    for (const row of state.lists[state.tab].rows) {
      if (row.node) row.node.hidden = !matchesSearch(row);
    }
    const sel = state.selected;
    if (sel && sel.tab === state.tab) {
      const row = state.lists[state.tab].rows.find((r) => r.key === sel.key);
      if (row && !matchesSearch(row)) clearDetail();
    }
    keepRowInTabOrder();
    updateCount();
  }

  function filtersSet() {
    const f = state.f;
    const shared = f.q || f.writeKey.length || f.since;
    if (state.tab === 'events') return shared || f.event.length || f.type.length || f.userId || f.anonymousId;
    return shared || f.kind !== 'all' || f.statusCode.length || f.failed || f.messageId;
  }

  function updateCount() {
    const rows = state.lists[state.tab].rows;
    const shown = rows.filter(matchesSearch).length;
    const s = state.summary && state.summary.summary;
    const events = state.tab === 'events';
    const total = s ? (events ? s.events.total : s.requests.total + s.control.total) : rows.length;
    const noun = events ? 'accepted events' : 'requests';
    $('count').textContent = filtersSet() || shown !== total ? `Showing ${shown} of ${total} ${noun}` : `${total} ${noun}`;

    const empty = $('empty');
    // A stopped page shows why in its own notice.
    empty.hidden = shown > 0 || !state.summary || Boolean(state.stopped);
    if (total === 0) {
      empty.textContent = events ? 'No events yet. They show here as your app sends them.' : 'No requests yet.';
    } else {
      empty.textContent = `Nothing matches the filters. The capture holds ${total} ${noun}.`;
    }
  }

  // ---- Selection and the detail pane.

  function rowsInView() {
    return [...$('list-body').querySelectorAll('tr:not([hidden])')];
  }

  function select(key, focus) {
    const tab = state.tab;
    const row = state.lists[tab].rows.find((r) => r.key === key);
    if (!row) return;
    state.selected = { tab, key };
    for (const tr of $('list-body').children) {
      const on = tr === row.node;
      tr.setAttribute('aria-selected', String(on));
      tr.tabIndex = on ? 0 : -1;
    }
    if (focus) row.node.focus();
    $('detail').classList.add('open');
    syncOverlay();
    if (tab === 'events') showEvent(row);
    else showRequest(row.obj.seq);
  }

  function clearDetail() {
    state.selected = null;
    state.detailToken++;
    const detail = $('detail');
    detail.classList.remove('open');
    syncOverlay();
    detail.replaceChildren(el('p', { class: 'placeholder', text: 'Select a row to see it in full.' }));
    $('list-body').querySelectorAll('tr[aria-selected="true"]').forEach((tr) => tr.setAttribute('aria-selected', 'false'));
  }

  // On a narrow screen the detail covers the page. The content behind it is
  // inert while it is open, so focus and arrow keys stay in the detail.
  const narrow = window.matchMedia('(max-width: 860px)');
  const overlay = () => narrow.matches && $('detail').classList.contains('open');

  function syncOverlay() {
    const on = overlay();
    document.querySelectorAll('body > :not(main):not(dialog):not(script), main > :not(#panel), #panel > :not(.split), .split > .list-wrap')
      .forEach((node) => { node.inert = on; });
  }

  // renderDetail replaces the detail and keeps focus in it when focus was
  // there, or when it covers the page.
  function renderDetail(...kids) {
    const detail = $('detail');
    const inside = detail.contains(document.activeElement);
    fill(detail, ...kids);
    if (overlay()) detail.querySelector('.close').focus();
    else if (inside) detail.focus();
  }

  function closeDetail() {
    const sel = state.selected;
    clearDetail();
    const row = sel && state.lists[sel.tab].rows.find((r) => r.key === sel.key);
    if (row && row.node) { row.node.tabIndex = 0; row.node.focus(); }
  }

  function facts(pairs) {
    return el('dl', { class: 'facts' }, pairs.filter(([, v]) => v !== '' && v != null)
      .flatMap(([k, v]) => [el('dt', { text: k }), el('dd', null, v instanceof Node ? v : str(v))]));
  }

  // Indenting deeper JSON would grow the text with the square of its depth.
  const PRETTY_DEPTH = 64;

  // pretty indents JSON text without parsing it, so numbers, key order and
  // duplicate keys stay as sent. It returns null for JSON nested deeper than
  // PRETTY_DEPTH.
  function pretty(raw) {
    let out = '';
    let depth = 0;
    let inString = false;
    let escaped = false;
    const newline = () => '\n' + '  '.repeat(depth);
    for (let i = 0; i < raw.length; i++) {
      const c = raw[i];
      if (inString) {
        out += c;
        if (escaped) escaped = false;
        else if (c === '\\') escaped = true;
        else if (c === '"') inString = false;
        continue;
      }
      if (c === '"') { inString = true; out += c; continue; }
      if (' \n\r\t'.includes(c)) continue;
      if (c === '{' || c === '[') {
        let j = i + 1;
        while (j < raw.length && ' \n\r\t'.includes(raw[j])) j++;
        if (raw[j] === (c === '{' ? '}' : ']')) { out += c + raw[j]; i = j; continue; }
        if (++depth > PRETTY_DEPTH) return null;
        out += c + newline();
        continue;
      }
      if (c === '}' || c === ']') { depth--; out += newline() + c; continue; }
      if (c === ',') { out += ',' + newline(); continue; }
      out += c === ':' ? ': ' : c;
    }
    return out;
  }

  const isJSON = (text) => { try { JSON.parse(text); return true; } catch { return false; } };

  // codeBlock shows text as sent, with a pretty form for JSON and a copy button.
  function codeBlock(title, raw, note) {
    const indented = isJSON(raw) ? pretty(raw) : null;
    const json = indented !== null;
    const pre = el('pre', { class: 'code', tabindex: '0', text: json ? indented : raw });
    const copy = el('button', { type: 'button', text: 'Copy', 'aria-label': `Copy ${title}` });
    copy.addEventListener('click', () => {
      navigator.clipboard.writeText(raw).then(() => { copy.textContent = 'Copied'; }, () => { copy.textContent = 'Copy failed'; });
      setTimeout(() => { copy.textContent = 'Copy'; }, 1500);
    });
    const toggle = json && el('button', { type: 'button', 'aria-pressed': 'false', text: 'Exact bytes' });
    if (toggle) {
      toggle.addEventListener('click', () => {
        const exact = toggle.getAttribute('aria-pressed') !== 'true';
        toggle.setAttribute('aria-pressed', String(exact));
        pre.textContent = exact ? raw : indented;
      });
    }
    return el('div', { class: 'block' },
      el('div', { class: 'block-head' }, el('h3', { text: title }), el('div', { class: 'actions' }, toggle, copy)),
      note && el('p', { class: 'note', text: note }),
      pre);
  }

  function detailHead(title, badge) {
    return el('div', { class: 'head' },
      el('h2', { text: title }),
      el('div', { class: 'actions' }, badge,
        el('button', { type: 'button', class: 'close', text: 'Close', onclick: closeDetail })));
  }

  function showEvent(row) {
    const token = ++state.detailToken;
    const o = row.obj || {};
    const id = typeof o.messageId === 'string' ? o.messageId : '';
    const request = el('div', null, el('p', { class: 'note', text: id ? 'Reading the request that carried it…' : '' }));
    renderDetail(
      detailHead(eventLabel(o) || str(o.type) || 'Event', o.type ? el('span', { class: 'badge info', text: str(o.type) }) : null),
      facts([['Type', o.type], ['Event', o.event], ['User ID', o.userId], ['Anonymous ID', o.anonymousId],
        ['Message ID', o.messageId], ['Sent (SDK clock)', timeOf(o)]]),
      codeBlock('Event as sent', row.line, 'The bytes the SDK sent for this event, shown in the view the filters ask for.'),
      el('h3', { text: 'Request' }),
      request,
    );
    if (!id) {
      request.replaceChildren(el('p', { class: 'note', text: 'This event has no message ID in this view, so its request is not linked. Show the whole event to link it.' }));
      return;
    }
    const p = new URLSearchParams({ serverId: state.serverId, messageId: id, kind: 'ingestion', view: 'full', maxBytes: '0', limit: '1' });
    api('requests', p).then((res) => res.text()).then((text) => {
      if (token !== state.detailToken) return;
      const env = parseExact(text);
      const rec = env.requests[0];
      if (!rec) {
        request.replaceChildren(el('p', { class: 'note', text: 'The listener no longer holds the request of this event.' }));
        return;
      }
      const mine = rec.events.find((e) => e.messageId === id) || {};
      fill(request,
        facts([['Request', `#${rec.seq}`], ['Received', rec.receivedAt], ['Route', rec.route], ['Status', statusBadge(rec.statusCode)],
          ['Write key', rec.writeKey === '' ? '(none)' : rec.writeKey], ['Events in request', rec.events.length],
          ['Same message ID', env.total > 1 ? `in ${env.total} requests` : '']]),
        el('p', null, el('button', { type: 'button', text: 'Open this request', onclick: () => openRequest(rec.seq, id) })),
        mine.enrichment ? codeBlock('What RudderStack adds', JSON.stringify(mine.enrichment),
          'RudderStack adds these keys when it receives the event. The list above shows the event without them.') : null,
      );
    }).catch((err) => {
      if (token === state.detailToken) request.replaceChildren(el('p', { class: 'note', text: `The request cannot be shown: ${err.message}` }));
    });
  }

  function openRequest(seq, messageId) {
    setFilters('requests', { messageId }).then(() => {
      const row = state.lists.requests.rows.find((r) => r.obj.seq === seq);
      if (row) select(row.key, true);
    });
  }

  // showRequest reads the whole record (maxBytes=0): a large batch is still
  // one record, and the pane shows it in full.
  function showRequest(seq) {
    const token = ++state.detailToken;
    renderDetail(detailHead(`Request #${seq}`), el('p', { class: 'note', text: 'Reading…' }));
    fetchRecord(seq, state.requestDetailView).then((rec) => {
      if (token !== state.detailToken) return;
      const full = state.requestDetailView === 'full';
      const viewToggle = el('button', {
        type: 'button', 'aria-pressed': String(full), text: 'Each event as sent',
        onclick: () => { state.requestDetailView = full ? 'compact' : 'full'; showRequest(seq); },
      });
      const rej = rec.rejection;
      renderDetail(
        detailHead(`Request #${rec.seq}`, statusBadge(rec.statusCode)),
        rej && el('div', { class: 'reason', role: 'note' },
          el('p', null, el('strong', { text: `Refused at the ${rej.stage} step` + (rej.idx != null ? ` (event ${rej.idx})` : '') + ': ' }), rej.reason)),
        facts([['Received', rec.receivedAt], ['Kind', rec.kind], ['Outcome', rec.outcome], ['Method', rec.request.method],
          ['Target', rec.request.target], ['Route', rec.route], ['Transport', rec.transport],
          ['Write key', rec.writeKey === '' ? '(none)' : rec.writeKey], ['Key SHA-256', rec.writeKeySha256], ['Source ID', rec.sourceId],
          ['From', rec.request.remoteAddr], ['Body size', `${rec.request.bodyBytes} bytes` + (rec.request.bodyComplete ? '' : ', cut short')],
          ['Headers left out', rec.request.droppedHeaders || ''], ['Hint', rec.hint]]),
        el('h3', { text: 'Request headers' }),
        facts(Object.entries(rec.request.headers || {}).map(([k, v]) => [k, v.join(', ')])),
        codeBlock('Request body', rec.request.body, 'The body as received, after gzip decoding.'),
        el('h3', { text: 'Response' }),
        facts([['Status', statusBadge(rec.response.statusCode)],
          ...Object.entries(rec.response.headers || {}).map(([k, v]) => [k, v.join(', ')])]),
        codeBlock('Response body', rec.response.body),
        el('div', { class: 'block-head' }, el('h3', { text: `Events (${rec.events.length})` }), el('div', { class: 'actions' }, viewToggle)),
        ...rec.events.map((e) => el('div', { class: 'event-block' },
          facts([['Event', `#${e.idx}`], ['Type', e.type], ['Name', e.event], ['Message ID', e.messageId],
            ['User ID', e.userId], ['Anonymous ID', e.anonymousId]]),
          e.enrichment && codeBlock(`What RudderStack adds to event ${e.idx}`, JSON.stringify(e.enrichment)),
          e.message && codeBlock(`Event ${e.idx}`, JSON.stringify(e.message),
            'Decoded and encoded again. The request body above holds the exact bytes.'))),
      );
    }).catch((err) => {
      if (token === state.detailToken) {
        renderDetail(detailHead(`Request #${seq}`), el('p', { class: 'note', text: `The request cannot be shown: ${err.message}` }));
      }
    });
  }

  // ---- Filters, tabs, export and keys.

  function setFilters(tab, values) {
    state.f = { ...DEFAULTS, ...values };
    state.tab = tab;
    syncTabs();
    fillForm();
    writeURL();
    return reload();
  }

  function syncTabs() {
    for (const name of ['events', 'requests']) {
      const tab = $('tab-' + name);
      const on = state.tab === name;
      tab.setAttribute('aria-selected', String(on));
      tab.tabIndex = on ? 0 : -1;
    }
    $('panel').setAttribute('aria-labelledby', 'tab-' + state.tab);
  }

  function switchTab(name) {
    if (state.tab === name) return;
    readForm();
    state.tab = name;
    syncTabs();
    fillForm();
    writeURL();
    clearDetail();
    renderRows();
  }

  let typing = 0;
  function onFilterChange(event) {
    const name = event.target.name;
    if (name === 'q') {
      state.f.q = event.target.value;
      clearTimeout(typing);
      typing = setTimeout(() => { writeURL(); applySearch(); }, 150);
      return;
    }
    if (event.type === 'input') return;
    if (name === 'since') {
      const custom = event.target.value === 'custom';
      showCustomSince(custom);
      // An empty custom field changes nothing until the person types in it.
      if (custom && form().elements.sinceAt.value.trim() === '') {
        form().elements.sinceAt.focus();
        return;
      }
    }
    readForm();
    writeURL();
    reload();
  }

  function download(name, type, text) {
    const url = URL.createObjectURL(new Blob([text], { type }));
    const a = el('a', { href: url, download: name });
    document.body.append(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 10000);
  }

  // exportShown saves what the list shows: events as NDJSON in the order
  // received, requests as the whole records in a JSON array.
  async function exportShown() {
    const stamp = `${state.serverId}-${state.cursor}`;
    const rows = state.lists[state.tab].rows.filter(matchesSearch);
    if (state.tab === 'events') {
      download(`events-${stamp}.ndjson`, 'application/x-ndjson', rows.map((r) => r.line + '\n').join(''));
      return;
    }
    const keys = new Set(rows.map((r) => r.key));
    const records = [];
    let since = state.f.since;
    try {
      for (;;) {
        const page = await fetchRequests(since, 'full');
        records.push(...page.rows.filter((r) => keys.has(r.key)).map((r) => r.obj));
        if (!page.hasMore) break;
        since = String(page.cursor);
      }
    } catch (err) {
      handleError(err);
      return;
    }
    const gone = keys.size - records.length;
    showNotice('export', 'warn', gone > 0 ?
      `The export leaves out ${plural(gone, 'listed request')}: the listener no longer holds ${gone === 1 ? 'it' : 'them'}.` : '');
    download(`requests-${stamp}.json`, 'application/json', JSON.stringify(records, null, 2) + '\n');
  }

  function onListKey(event) {
    const rows = rowsInView();
    if (document.activeElement === $('list')) {
      // The skip link lands on the list itself; these keys enter the rows.
      const first = { ArrowDown: rows[0], Home: rows[0], End: rows[rows.length - 1] }[event.key];
      if (!first) return;
      event.preventDefault();
      rows.forEach((tr) => { tr.tabIndex = tr === first ? 0 : -1; });
      first.focus();
      return;
    }
    const i = rows.indexOf(document.activeElement);
    if (i < 0) return;
    let next = null;
    if (event.key === 'ArrowDown') next = rows[i + 1];
    else if (event.key === 'ArrowUp') next = rows[i - 1];
    else if (event.key === 'Home') next = rows[0];
    else if (event.key === 'End') next = rows[rows.length - 1];
    else if (event.key === 'Enter' || event.key === ' ') next = rows[i];
    else return;
    event.preventDefault();
    if (next) select(Number(next.dataset.key), true);
  }

  function onGlobalKey(event) {
    const typingIn = /^(INPUT|SELECT|TEXTAREA)$/.test(document.activeElement.tagName);
    if (event.key === '/' && !typingIn && !$('help').open) {
      event.preventDefault();
      form().elements.q.focus();
    } else if (event.key === 'Escape' && state.selected && !$('help').open) {
      closeDetail();
    }
  }

  function onTabKey(event) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    const name = state.tab === 'events' ? 'requests' : 'events';
    switchTab(name);
    $('tab-' + name).focus();
  }

  function bind() {
    const f = form();
    f.addEventListener('input', onFilterChange);
    f.addEventListener('change', onFilterChange);
    f.addEventListener('submit', (e) => e.preventDefault());
    $('only-new').addEventListener('click', () => {
      state.f.since = String(state.cursor);
      showSince();
      writeURL();
      reload();
    });
    $('tab-events').addEventListener('click', () => switchTab('events'));
    $('tab-requests').addEventListener('click', () => switchTab('requests'));
    $('tab-events').addEventListener('keydown', onTabKey);
    $('tab-requests').addEventListener('keydown', onTabKey);
    $('list').addEventListener('keydown', onListKey);
    $('pause').addEventListener('click', togglePause);
    $('export').addEventListener('click', exportShown);
    $('help-open').addEventListener('click', () => $('help').showModal());
    $('help-close').addEventListener('click', () => $('help').close());
    document.addEventListener('keydown', onGlobalKey);
    narrow.addEventListener('change', syncOverlay);
  }

  async function start() {
    readURL();
    syncTabs();
    fillForm();
    bind();
    let info;
    for (;;) {
      try {
        info = await (await api('info')).json();
        break;
      } catch (err) {
        if (!handleError(err)) return;
        await sleep(RETRY_MS);
      }
    }
    state.serverId = info.serverId;
    state.url = info.url;
    $('where').textContent = `${info.url} · started ${clock(info.startedAt)}`;
    $('help-url').textContent = info.url;
    if (info.exposed) {
      showNotice('exposed', 'warn', 'This listener takes connections from other machines. Anyone who reaches it can read this page.');
    }
    setLive('live');
    reload();
  }

  start();
})();
