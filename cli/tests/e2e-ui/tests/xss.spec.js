// Any web page can post to a local listener, so every string the review page
// shows is attacker-controlled. This test puts a hostile payload into every
// rendered field a sender controls and fails if any of them runs, becomes
// markup or makes the page fetch something.
const fs = require('node:fs');
const { test, expect, open } = require('./fixtures');

const PAYLOADS = [
  '<img src=x onerror="window.__pwned=1">',
  '"><svg onload="window.__pwned=1">',
  "'><script>window.__pwned=1</script>",
  '</script><script>window.__pwned=1</script>',
  '<a href="javascript:window.__pwned=1" id="evil-link">x</a>',
  '${window.__pwned=1}',
  '{{constructor.constructor("window.__pwned=1")()}}',
  // The first payload's image loads /_dev/ui/x, inside the page's own
  // prefix. This one would load a path outside it, which the test catches.
  '<img src="/pwned">',
];

async function postHostile(listener, payload) {
  const event = {
    type: 'track',
    event: payload,
    userId: payload,
    anonymousId: payload,
    messageId: 'xss-' + Math.random().toString(16).slice(2),
    properties: { [payload]: payload, nested: { list: [payload] } },
    context: { library: { name: payload, version: payload }, page: { path: payload, url: payload } },
  };
  // The request detail renders the kept headers, and AnonymousId is one.
  const headers = { AnonymousId: payload };
  // The listener stores a batch event of any type, and the page shows the
  // type in the Type column, the badge and the detail.
  await listener.send('/v1/batch', { batch: [{ ...event, type: payload }] }, payload, headers);
  await listener.send('/v1/track?x=' + encodeURIComponent(payload), event, payload, headers);
  // A refused request keeps the payload in its body and header. The rejection
  // reason is a fixed string and the target is the plain /v1/track.
  await listener.send('/v1/track', '{"event":' + JSON.stringify(payload) + ',"userId":', 'dev', headers);
}

test.describe('hostile events', () => {
  test('nothing a page posts runs in the review page', async ({ page, listener }) => {
    for (const payload of PAYLOADS) await postHostile(listener, payload);

    let dialog = null;
    page.on('dialog', async (d) => { dialog = d.message(); await d.dismiss(); });
    const outside = [];
    page.on('request', (req) => {
      const u = new URL(req.url());
      if (u.origin !== new URL(listener.url).origin || !u.pathname.startsWith('/_dev/')) outside.push(req.url());
    });
    await open(page, listener);

    // Lists show the newest row first, so opening only the top row would skip
    // most payloads. Open every row of both tabs.
    for (const tab of ['Events', 'Requests']) {
      await page.getByRole('tab', { name: tab }).click();
      const rows = page.locator('#list-body tr:not([hidden])');
      await expect(rows.first()).toBeVisible();
      const count = await rows.count();
      expect(count).toBeGreaterThanOrEqual(PAYLOADS.length * 2);
      for (let i = 0; i < count; i++) {
        await rows.nth(i).click();
        // Both details load the request async and drop the result when the
        // next click lands. The Events tab shows "Reading the request that
        // carried it…" and the Requests tab "Reading…". Wait for each load, so
        // the checks below run after the last detail has rendered.
        await expect(page.locator('#detail').getByText(/^Reading/)).toHaveCount(0);
      }
    }

    expect(dialog, 'no alert, confirm or prompt').toBeNull();
    expect(await page.evaluate(() => window.__pwned), 'payload script ran').toBeUndefined();
    // Payload text may appear, but only as text, never as elements. The page
    // itself renders no image, SVG or script inside main.
    await expect(page.locator('#evil-link')).toHaveCount(0);
    await expect(page.locator('main img, main svg, main script')).toHaveCount(0);
    expect(outside, 'requests outside /_dev/').toEqual([]);
  });

  test('the export is plain text a browser will not render', async ({ page, listener }) => {
    await postHostile(listener, PAYLOADS[0]);
    await open(page, listener);
    await page.getByRole('tab', { name: 'Events' }).click();
    const [file] = await Promise.all([
      page.waitForEvent('download'),
      page.getByRole('button', { name: 'Export' }).click(),
    ]);
    // A .html or .svg name would make the browser render the payload.
    expect(file.suggestedFilename()).toMatch(/^[\w.-]+\.ndjson$/);
    const text = fs.readFileSync(await file.path(), 'utf8');
    // Each line is one event as JSON. The payload is a string value in it,
    // never a tag at the start of the file.
    expect(text.startsWith('{')).toBe(true);
    const events = text.trim().split('\n').map((line) => JSON.parse(line));
    expect(events.map((e) => e.event)).toContain(PAYLOADS[0]);
  });
});
