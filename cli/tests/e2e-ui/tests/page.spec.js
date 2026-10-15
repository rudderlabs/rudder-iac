// Acceptance and regression tests of the review page at /_dev/ui/. Each test
// drives the built binary through a real browser.
const fs = require('node:fs');
const { test, expect, seed, open, apiRequests, lastList } = require('./fixtures');

const rows = (page) => page.locator('#list-body tr:not([hidden])');

// setText types into a field and leaves it, which commits the change.
async function setText(page, label, value) {
  const field = page.getByLabel(label, { exact: true });
  await field.fill(value);
  await field.press('Tab');
}

async function download(page) {
  const [file] = await Promise.all([page.waitForEvent('download'), page.getByRole('button', { name: 'Export' }).click()]);
  return fs.readFileSync(await file.path(), 'utf8');
}

async function sendBatches(listener, batches, size) {
  for (let b = 0; b < batches; b++) {
    const batch = Array.from({ length: size }, (_, i) => ({ type: 'track', event: 'Bulk', userId: 'u', messageId: `bulk-${b}-${i}` }));
    expect((await listener.send('/v1/batch', { batch })).status).toBe(200);
  }
}

async function summary(listener) {
  return (await (await listener.api('events?view=counts')).json()).summary;
}

async function eventLines(listener, query) {
  const text = await (await listener.api('events?limit=1000&' + query)).text();
  return text.split('\n').filter((line) => line !== '');
}

test.describe('acceptance', () => {
  test('tiles show the whole capture and a filtered list reads N of M', async ({ page, listener }) => {
    await seed(listener);
    await open(page, listener);
    const s = await summary(listener);
    await expect(page.locator('#count')).toHaveText(`${s.events.total} accepted events`);
    const num = (i) => page.locator('#tiles .tile .num').nth(i);
    await expect(num(0)).toHaveText(String(s.requests.total));
    await expect(num(1)).toHaveText(String(s.events.total));
    await expect(num(2)).toHaveText(String(s.rejected.events));
    await expect(num(5)).toHaveText(String(Object.keys(s.byWriteKey).length));
    await page.locator('#breakdown summary').click();
    await expect(page.locator('#by-event')).toContainText('Plan Selected');
    await expect(page.locator('#by-rejected')).toContainText('Order Completed');
    await expect(page.locator('#by-key li')).toHaveCount(2);
    await expect(page.locator('#by-key')).toContainText('1 request, 1 event');
    await expect(page.locator('#list-head')).toContainText('Sent (SDK clock)');

    await setText(page, 'Event name', 'Order*');
    const matched = await eventLines(listener, 'event=Order*');
    await expect(page.locator('#count')).toHaveText(`Showing ${matched.length} of ${s.events.total} accepted events`);
    await expect(num(1)).toHaveText(String(s.events.total));
  });

  test('the events tab lists accepted events only, as many as the API for the same filters', async ({ page, listener }) => {
    await seed(listener);
    await open(page, listener);
    await page.getByLabel('track', { exact: true }).check();
    await setText(page, 'Write keys', 'dev');
    const lines = await eventLines(listener, 'type=track&writeKey=dev');
    expect(lines.length).toBeGreaterThan(0);
    await expect(page.locator('#count')).toHaveText(new RegExp(`^Showing ${lines.length} of `));
    await expect(rows(page)).toHaveCount(lines.length);
    await expect(page.locator('#list-body')).not.toContainText('m-bad');
  });

  test('the requests tab lists every request; the detail shows reason, headers, body, response and enrichment', async ({ page, listener }) => {
    await seed(listener);
    await open(page, listener);
    await page.getByRole('tab', { name: 'Requests' }).click();
    const env = await (await listener.api('requests?kind=all&view=compact&limit=1000&maxBytes=0')).json();
    await expect(rows(page)).toHaveCount(env.total);

    const refused = env.requests.find((r) => r.statusCode === 400 && r.rejection.stage !== 'body');
    const row = page.locator(`#list-body tr[data-key="${refused.seq}"]`);
    await expect(row).toContainText('400');
    await expect(row).toContainText(refused.rejection.reason);
    await row.click();
    const detail = page.locator('#detail');
    await expect(detail.locator('.reason')).toContainText(`Refused at the ${refused.rejection.stage} step`);
    const record = await (await listener.api(`requests/${refused.seq}?maxBytes=0`)).json();
    const headers = detail.locator('h3:text-is("Request headers") + dl dt');
    await expect(headers).toHaveCount(Object.keys(record.request.headers).length);
    await expect(detail.locator('.block', { hasText: 'Request body' }).locator('pre')).toContainText('"m-bad"');
    await expect(detail.locator('.block', { hasText: 'Response body' }).locator('pre')).toContainText(record.response.body.trim().slice(0, 20));

    await page.locator('#list-body tr[data-key="1"]').click();
    await expect(detail).toContainText('What RudderStack adds to event 0');
    await expect(detail).toContainText('Events (5)');
  });

  test('the write-key filter takes literal keys, several at once', async ({ page, listener }) => {
    await seed(listener);
    await open(page, listener);
    const sent = apiRequests(page, 'events');
    await setText(page, 'Write keys', 'dev, web-key-0123456789');
    await expect(page.locator('#count')).toHaveText('Showing 6 of 6 accepted events');
    await expect.poll(() => lastList(sent)?.getAll('writeKey')).toEqual(['dev', 'web-key-0123456789']);
    await setText(page, 'Write keys', 'web-key-0123456789');
    await expect(page.locator('#count')).toHaveText('Showing 1 of 6 accepted events');
    await setText(page, 'Write keys', '(none)');
    await expect(page.locator('#count')).toHaveText('Showing 0 of 6 accepted events');
    await expect.poll(() => lastList(sent)?.getAll('writeKey')).toEqual(['']);
  });

  test('a value set by a click keeps its comma when another filter changes', async ({ page, listener }) => {
    await listener.send('/v1/track', { type: 'track', event: 'Checkout, completed', userId: 'u' });
    await listener.send('/v1/track', { type: 'track', event: 'Checkout', userId: 'u' });
    await open(page, listener);
    const sent = apiRequests(page, 'events');
    await page.locator('#breakdown summary').click();
    await page.locator('#by-event').getByRole('button', { name: 'Checkout, completed' }).click();
    await expect(page.locator('#count')).toHaveText('Showing 1 of 2 accepted events');
    await page.getByLabel('track', { exact: true }).check();
    await expect(page.locator('#count')).toHaveText('Showing 1 of 2 accepted events');
    await expect.poll(() => lastList(sent)?.getAll('event')).toEqual(['Checkout, completed']);

    await setText(page, 'Event name', 'Checkout\\, completed, Nope');
    await expect(page.locator('#count')).toHaveText('Showing 1 of 2 accepted events');
    await expect.poll(() => lastList(sent)?.getAll('event')).toEqual(['Checkout, completed', 'Nope']);
  });

  test('new events show live, and Pause aborts the poll in flight', async ({ page, listener }) => {
    await open(page, listener);
    await listener.send('/v1/track', { type: 'track', event: 'Arrived Live', userId: 'u' });
    await expect(page.locator('#list-body')).toContainText('Arrived Live');

    const aborted = page.waitForEvent('requestfailed', (req) => req.url().includes('wait=5s'));
    await page.getByRole('button', { name: 'Pause' }).click();
    expect((await aborted).failure().errorText).toContain('ERR_ABORTED');
    await expect(page.locator('#live-text')).toHaveText('Paused');
    await listener.send('/v1/track', { type: 'track', event: 'While Paused', userId: 'u' });
    await page.waitForTimeout(1500);
    await expect(page.locator('#list-body')).not.toContainText('While Paused');
    await page.getByRole('button', { name: 'Resume' }).click();
    await expect(page.locator('#list-body')).toContainText('While Paused');
  });

  test('a bad filter shows text and stops asking', async ({ page, listener, consoleErrors }) => {
    await seed(listener);
    const sent = apiRequests(page, 'events');
    await page.goto(listener.ui + '?since=abc');
    await expect(page.locator('#filter-error')).toContainText('The Received filter is not valid');
    await expect(page.locator('#live-text')).toHaveText('Not updating');
    await expect(page.locator('#empty')).toBeHidden();
    const asked = sent.length;
    await page.waitForTimeout(6000);
    expect(sent.length).toBe(asked);
    expect(consoleErrors.every((text) => text.includes('400'))).toBe(true);
    consoleErrors.length = 0;
  });

  test('a stopped listener shows text', async ({ page, listener, consoleErrors }) => {
    await open(page, listener);
    await listener.stop();
    await expect(page.locator('#live-text')).toHaveText('Not updating');
    await expect(page.locator('#notices')).toContainText(/stopping|cannot be reached/);
    consoleErrors.length = 0;
  });

  test('a link from another site cannot set serverId or the pause state', async ({ page, listener }) => {
    await seed(listener);
    const sent = apiRequests(page, 'events');
    await open(page, listener, '?serverId=bogus&paused=true');
    await expect(page.getByRole('button', { name: 'Pause' })).toHaveAttribute('aria-pressed', 'false');
    await expect(page.locator('#count')).toHaveText('6 accepted events');
    expect(sent.length).toBeGreaterThan(0);
    for (const query of sent) expect(query.get('serverId')).toBe(listener.ready.serverId);
  });

  test('search matches property names and values', async ({ page, listener }) => {
    await seed(listener);
    await open(page, listener);
    await expect(page.locator('#count')).toHaveText('6 accepted events');
    const search = page.getByLabel('Search');
    await search.fill('currency');
    await expect(rows(page)).toHaveCount(1);
    await expect(rows(page)).toContainText('Order Completed');
    await search.fill('USD');
    await expect(rows(page)).toHaveCount(1);
    await search.fill('pricing');
    await expect(rows(page)).toHaveCount(2);
  });

  test('the page lists past 1,000 rows', async ({ page, listener }) => {
    await sendBatches(listener, 12, 100);
    const sent = apiRequests(page, 'events');
    await open(page, listener);
    await expect(page.locator('#count')).toHaveText('1200 accepted events');
    await expect(page.locator('#list-body tr')).toHaveCount(1200);
    expect(sent.filter((q) => !q.has('wait') && q.get('view') !== 'counts').length).toBeGreaterThan(1);
  });

  test('the page names no command', async ({ listener }) => {
    for (const file of ['', 'app.js', 'app.css']) {
      const text = await (await fetch(listener.ui + file)).text();
      expect(text.toLowerCase()).not.toContain('rudder-cli');
    }
  });
});

test.describe('regressions', () => {
  test('a timestamp object does not stop the page', async ({ page, listener }) => {
    await open(page, listener);
    await listener.send('/v1/track', { type: 'track', event: 'tsobj', userId: 'u', messageId: 'm-ts', originalTimestamp: { toString: 0 } });
    await listener.send('/v1/track', { type: 'track', event: 'tsobj2', userId: 'u', messageId: 'm-ts2', timestamp: { valueOf: 0, toString: 0 } });
    await expect(page.locator('#list-body')).toContainText('tsobj2');
    await expect(page.locator('#list-body')).toContainText('tsobj');

    await open(page, listener);
    await expect(page.locator('#count')).toHaveText('2 accepted events');
    await expect(page.locator('#notices')).not.toContainText('failed');
    await rows(page).filter({ hasText: 'tsobj2' }).click();
    await expect(page.locator('#detail h2')).toHaveText('tsobj2');
    await expect(page.locator('#live-text')).toHaveText('Live');
  });

  test('the detail of a 30-event batch shows every event', async ({ page, listener }) => {
    const batch = Array.from({ length: 30 }, (_, i) => ({
      type: 'track', event: `Item ${i}`, userId: 'u', messageId: `big-${i}`, properties: { blob: 'x'.repeat(1000) },
    }));
    expect((await listener.send('/v1/batch', { batch })).status).toBe(200);
    await open(page, listener);
    await page.getByRole('tab', { name: 'Requests' }).click();
    await page.locator('#list-body tr[data-key="1"]').click();
    await expect(page.locator('#detail')).toContainText('Events (30)');
    await expect(page.locator('#detail')).not.toContainText('Reading…');
  });

  test('the request detail shows no stray null', async ({ page, listener }) => {
    await seed(listener);
    await open(page, listener);
    await page.getByRole('tab', { name: 'Requests' }).click();
    await page.locator('#list-body tr[data-key="1"]').click();
    await expect(page.locator('#detail')).toContainText('Events (5)');
    expect(await page.locator('#detail').innerText()).not.toMatch(/^null$/m);
    await page.getByRole('tab', { name: 'Events' }).click();
    await rows(page).filter({ hasText: 'Signup Started' }).click();
    await expect(page.locator('#detail')).toContainText('Open this request');
    expect(await page.locator('#detail').innerText()).not.toMatch(/^null$/m);
  });

  test('a live update during a reload adds no row twice', async ({ page, listener }) => {
    // A page never splits a request, so the first page ends after ten
    // batches and the reload needs a second page.
    await sendBatches(listener, 11, 100);
    await open(page, listener);
    await expect(page.locator('#count')).toHaveText('1100 accepted events');

    // Hold the second page of the filtered reload, and send one event while
    // it is held. The live update then runs while the reload is unfinished.
    let held = false;
    let release;
    const released = new Promise((resolve) => { release = resolve; });
    await page.route('**/_dev/v1/events?**', async (route) => {
      const q = new URL(route.request().url()).searchParams;
      if (held || q.get('userId') !== 'u' || !q.has('since') || q.has('wait')) return route.continue();
      held = true;
      await listener.send('/v1/track', { type: 'track', event: 'Arrived During Reload', userId: 'u', messageId: 'live-1' });
      await new Promise((resolve) => setTimeout(resolve, 3000));
      await route.continue();
      return release();
    });
    await setText(page, 'User ID', 'u');
    await released;
    // Let the held page and anything queued behind it land.
    await page.waitForTimeout(2000);
    await expect(page.locator('#count')).toHaveText('Showing 1101 of 1101 accepted events');
    await expect(page.locator('#list-body tr')).toHaveCount(1101);
    const lines = (await download(page)).trim().split('\n');
    expect(lines).toHaveLength(1101);
    expect(new Set(lines).size).toBe(1101);
  });

  test('numbers above 2^53 keep their digits in the exports and the detail', async ({ page, listener }) => {
    await seed(listener);
    await open(page, listener);
    await expect(page.locator('#count')).toHaveText('6 accepted events');
    expect(await download(page)).toContain('"orderId":9007199254740993');

    await page.getByRole('tab', { name: 'Requests' }).click();
    await expect(page.locator('#list-body tr[data-key="1"]')).toBeVisible();
    const records = await download(page);
    expect(records).toContain('"orderId": 9007199254740993');
    expect(records).not.toContain('9007199254740992');

    await page.locator('#list-body tr[data-key="1"]').click();
    await page.getByRole('button', { name: 'Each event as sent' }).click();
    await expect(page.locator('#detail')).toContainText('Event 2');
    await expect(page.locator('#detail')).toContainText('9007199254740993');
    await expect(page.locator('#detail')).not.toContainText('9007199254740992');
  });

  test('after a search the keyboard reaches a row', async ({ page, listener }) => {
    await seed(listener);
    await open(page, listener);
    await expect(page.locator('#count')).toHaveText('6 accepted events');
    await page.getByLabel('Search').fill('pricing');
    await expect(rows(page)).toHaveCount(2);

    await page.locator('details.shape > summary').focus();
    await page.keyboard.press('Tab');
    const focused = () => page.evaluate(() => {
      const node = document.activeElement;
      return node.tagName === 'TR' && !node.hidden ? node.dataset.key : `${node.tagName}#${node.id}`;
    });
    const first = await focused();
    expect(first).toBe(await rows(page).first().getAttribute('data-key'));
    await page.keyboard.press('ArrowDown');
    expect(await focused()).toBe(await rows(page).nth(1).getAttribute('data-key'));
    await expect(page.locator('#detail')).toHaveClass(/open/);

    await page.keyboard.press('Escape');
    await page.locator('.skip').focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('#list')).toBeFocused();
    await page.keyboard.press('ArrowDown');
    expect(await focused()).toBe(first);
  });

  test('an event key nested near the depth limit does not stop the page', async ({ page, listener }) => {
    await open(page, listener);
    const depth = 9998;
    const body = '{"type":"track","userId":"u","messageId":"deep","event":' + '['.repeat(depth) + ']'.repeat(depth) + '}';
    expect((await listener.send('/v1/track', body)).status).toBe(200);
    await listener.send('/v1/track', { type: 'track', event: 'After Deep', userId: 'u' });
    await expect(page.locator('#list-body')).toContainText('After Deep');

    await page.getByRole('tab', { name: 'Requests' }).click();
    await expect(page.locator('#list-body tr')).toHaveCount(2);
    await page.locator('#list-body tr[data-key="1"]').click();
    await expect(page.locator('#detail h2')).toHaveText('Request #1');
    await expect(page.locator('#detail')).toContainText('Events (1)');
    await expect(page.locator('#notices')).not.toContainText('failed');
    await expect(page.locator('#live-text')).toHaveText('Live');
  });

  test('evicted requests leave the list and the export says so', async ({ page, listener }) => {
    test.setTimeout(180_000);
    await listener.send('/v1/track', { type: 'track', event: 'First', userId: 'u' });
    await open(page, listener);
    await page.getByRole('tab', { name: 'Requests' }).click();
    await expect(page.locator('#list-body tr')).toHaveCount(1);
    await page.getByRole('button', { name: 'Pause' }).click();

    // The listener keeps the newest 10,000 requests.
    for (let sent = 0; sent < 10_000; sent += 50) {
      await Promise.all(Array.from({ length: 50 }, (_, i) =>
        listener.send('/v1/track', { type: 'track', event: 'Fill', userId: 'u', messageId: `fill-${sent + i}` })));
    }
    await page.getByRole('button', { name: 'Resume' }).click();
    await expect(page.locator('#notices')).toContainText('The requests list no longer shows them');
    await expect(page.locator('#list-body tr')).toHaveCount(10_000);
    await expect(page.locator('#list-body tr[data-key="1"]')).toHaveCount(0);
  });

  test('on a narrow screen the open detail holds focus', async ({ page, listener }) => {
    await page.setViewportSize({ width: 400, height: 800 });
    await seed(listener);
    await open(page, listener);
    await expect(page.locator('#count')).toHaveText('6 accepted events');
    const row = rows(page).first();
    await row.focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('#detail .close')).toBeFocused();
    expect(await page.evaluate(() => document.querySelector('.list-wrap').inert)).toBe(true);
    await page.keyboard.press('Escape');
    await expect(row).toBeFocused();
    expect(await page.evaluate(() => document.querySelector('.list-wrap').inert)).toBe(false);
  });
});
