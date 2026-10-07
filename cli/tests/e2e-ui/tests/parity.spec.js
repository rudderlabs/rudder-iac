// Parity: every parameter of /_local/v1/events and /_local/v1/requests has a
// control on the page, and setting the control changes the query the page
// sends. The parameter lists come from the API index, so a new parameter
// fails this test until the page has a control for it.
const { test, expect, seed, open } = require('./fixtures');

// The page sets these itself; no control changes them.
const PAGE_SET = {
  serverId: 'from /info on every request, never from the page URL',
  wait: 'the live poll',
  min: 'the server default of 1',
  limit: 'fixed: every list loads whole',
  maxBytes: 'fixed to 0: the page reads whole records',
};

async function setText(page, label, value) {
  const field = page.getByLabel(label, { exact: true });
  await field.fill(value);
  await field.press('Tab');
}

async function showColumns(page) {
  const shape = page.locator('details.shape');
  if (!(await shape.getAttribute('open'))) await shape.locator('summary').click();
}

const shared = {
  since: {
    set: async (page) => {
      await page.locator('select[name="since"]').selectOption('custom');
      await setText(page, 'Received after', '2026-10-01T10:00:00Z');
    },
    sent: (q) => q.get('since') === '2026-10-01T10:00:00Z',
  },
  writeKey: { set: (page) => setText(page, 'Write keys', 'dev'), sent: (q) => q.getAll('writeKey').join() === 'dev' },
};

const CONTROLS = {
  events: {
    ...shared,
    event: { set: (page) => setText(page, 'Event name', 'Order*'), sent: (q) => q.getAll('event').join() === 'Order*' },
    type: { set: (page) => page.getByLabel('page', { exact: true }).check(), sent: (q) => q.getAll('type').join() === 'page' },
    userId: { set: (page) => setText(page, 'User ID', 'user-1042'), sent: (q) => q.get('userId') === 'user-1042' },
    anonymousId: { set: (page) => setText(page, 'Anonymous ID', 'anon-9'), sent: (q) => q.get('anonymousId') === 'anon-9' },
    view: {
      set: async (page) => { await showColumns(page); await page.getByLabel('Show').selectOption('compact'); },
      sent: (q) => q.get('view') === 'compact',
    },
    fields: {
      set: async (page) => { await showColumns(page); await setText(page, 'Only these fields', 'properties'); },
      sent: (q) => q.getAll('fields').join() === 'properties',
    },
  },
  requests: {
    ...shared,
    kind: {
      set: (page) => page.getByLabel('Kind').selectOption('control'),
      sent: (q) => q.get('kind') === 'control',
    },
    statusCode: { set: (page) => setText(page, 'Status', '4xx'), sent: (q) => q.getAll('statusCode').join() === '4xx' },
    failed: { set: (page) => page.getByLabel('Failed only').check(), sent: (q) => q.get('failed') === 'true' },
    messageId: { set: (page) => setText(page, 'Message ID', 'm-1'), sent: (q) => q.get('messageId') === 'm-1' },
    view: {
      set: async (page) => { await showColumns(page); await page.getByLabel('Show').selectOption('full'); },
      sent: (q) => q.get('view') === 'full',
    },
    fields: {
      set: async (page) => { await showColumns(page); await setText(page, 'Only these fields', 'rejection'); },
      sent: (q) => q.getAll('fields').join() === 'rejection',
    },
  },
};

// A list load is a request of the tab's route that is not the live poll and
// not the summary.
const isListLoad = (route) => (req) => {
  const url = new URL(req.url());
  return url.pathname === '/_local/v1/' + route && !url.searchParams.has('wait') && url.searchParams.get('view') !== 'counts';
};

async function endpointParams(listener, route) {
  const index = await (await listener.api('')).json();
  const endpoint = index.endpoints.find((e) => e.path === '/_local/v1/' + route);
  expect(endpoint, `the index lists /_local/v1/${route}`).toBeTruthy();
  return endpoint.params;
}

for (const route of ['events', 'requests']) {
  test.describe(`/_local/v1/${route}`, () => {
    test('every parameter has a control or is set by the page', async ({ listener }) => {
      const params = await endpointParams(listener, route);
      expect(params.length).toBeGreaterThan(0);
      const missing = params.filter((p) => !PAGE_SET[p] && !CONTROLS[route][p]);
      expect(missing, 'parameters with no page control').toEqual([]);
      const stale = Object.keys(CONTROLS[route]).filter((p) => !params.includes(p));
      expect(stale, 'controls for parameters the API does not take').toEqual([]);
    });

    for (const [param, control] of Object.entries(CONTROLS[route])) {
      test(`setting ${param} changes the query`, async ({ page, listener }) => {
        await seed(listener);
        const loads = [];
        page.on('request', (req) => { if (isListLoad(route)(req)) loads.push(new URL(req.url()).searchParams); });
        await open(page, listener);
        if (route === 'requests') await page.getByRole('tab', { name: 'Requests' }).click();
        await expect.poll(() => loads.length).toBeGreaterThan(0);
        expect(control.sent(loads.at(-1)), `${param} before the change`).toBe(false);

        const next = page.waitForRequest((req) => isListLoad(route)(req) && control.sent(new URL(req.url()).searchParams));
        await control.set(page);
        await next;
      });
    }

    test('the page sets its own parameters', async ({ page, listener }) => {
      await seed(listener);
      const loads = [];
      page.on('request', (req) => { if (isListLoad(route)(req)) loads.push(new URL(req.url()).searchParams); });
      await open(page, listener);
      await expect.poll(() => loads.length).toBeGreaterThan(0);
      for (const q of loads) {
        expect(q.get('serverId')).toBe(listener.ready.serverId);
        expect(q.get('limit')).toBe('1000');
        if (route === 'requests') expect(q.get('maxBytes')).toBe('0');
        expect(q.has('min')).toBe(false);
      }
    });
  });
}

test('the preset times send a duration', async ({ page, listener }) => {
  await open(page, listener);
  const next = page.waitForRequest((req) => isListLoad('events')(req) && new URL(req.url()).searchParams.get('since') === '5m');
  await page.locator('select[name="since"]').selectOption('5m');
  await next;
});

test('Only new sends the current cursor and shows it in the custom field', async ({ page, listener }) => {
  await seed(listener);
  await open(page, listener);
  const cursor = String((await (await listener.api('events?view=counts')).json()).cursor);
  const next = page.waitForRequest((req) => isListLoad('events')(req) && new URL(req.url()).searchParams.get('since') === cursor);
  await page.getByRole('button', { name: 'Only new' }).click();
  await next;
  await expect(page.getByLabel('Received after')).toHaveValue(cursor);
});
