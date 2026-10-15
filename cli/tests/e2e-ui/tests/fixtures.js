// Fixtures for the review page tests. Each test gets its own listener, so
// tests run in parallel and never share a capture.
const { test: base, expect } = require('@playwright/test');
const { spawn } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const readline = require('node:readline');

const bin = process.env.RUDDER_CLI_BIN || path.resolve(__dirname, '../../../../bin/rudder-cli');

async function startListener() {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'dev-listen-page-'));
  const child = spawn(bin, ['dev', 'listen'], {
    env: {
      ...process.env,
      HOME: home,
      RUDDERSTACK_CLI_EXPERIMENTAL: 'true',
      RUDDERSTACK_X_DEV_LISTEN: 'true',
      RUDDERSTACK_CLI_TELEMETRY_DISABLED: 'true',
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  let stderr = '';
  child.stderr.on('data', (chunk) => { stderr += chunk; });
  const exited = new Promise((resolve) => child.once('exit', resolve));
  const line = await Promise.race([
    new Promise((resolve) => readline.createInterface({ input: child.stdout }).once('line', resolve)),
    exited.then(() => { throw new Error(`dev listen exited before its ready line: ${stderr}`); }),
    new Promise((_, reject) => setTimeout(() => reject(new Error(`no ready line: ${stderr}`)), 15_000)),
  ]);
  const ready = JSON.parse(line);

  return {
    ready,
    url: ready.url,
    ui: ready.ui,
    // send posts one SDK request with a basic-auth write key.
    async send(route, body, key = 'dev') {
      const res = await fetch(ready.url + route, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'User-Agent': 'Mozilla/5.0 page-test',
          Authorization: 'Basic ' + Buffer.from(key + ':').toString('base64'),
        },
        body: typeof body === 'string' ? body : JSON.stringify(body),
      });
      return { status: res.status, text: await res.text() };
    },
    async api(route) {
      const res = await fetch(ready.url + '/_dev/v1/' + route);
      if (!res.ok) throw new Error(`${route}: ${res.status} ${await res.text()}`);
      return res;
    },
    async stop() {
      if (child.exitCode === null && child.signalCode === null) {
        child.kill('SIGTERM');
        await exited;
      }
      fs.rmSync(home, { recursive: true, force: true });
    },
  };
}

// seed sends the capture the screenshots and most tests use: a batch of
// five events, a track with another key, two refused requests and one
// settings request.
async function seed(listener) {
  const ctx = { library: { name: 'RudderLabs JavaScript SDK', version: '3.11.0' } };
  await listener.send('/v1/batch', `{"batch":[
{"type":"identify","userId":"user-1042","anonymousId":"anon-7f3a","messageId":"m-1","originalTimestamp":"2026-10-01T17:55:00.000Z","traits":{"plan":"pro","email":"a@example.com"},"context":{"library":{"name":"RudderLabs JavaScript SDK","version":"3.11.0"},"page":{"path":"/pricing"}}},
{"type":"page","name":"Pricing","userId":"user-1042","anonymousId":"anon-7f3a","messageId":"m-2","originalTimestamp":"2026-10-01T17:55:01.000Z","properties":{"path":"/pricing","title":"Pricing"},"context":{"library":{"name":"RudderLabs JavaScript SDK"},"page":{"path":"/pricing"}}},
{"type":"track","event":"Plan Selected","userId":"user-1042","anonymousId":"anon-7f3a","messageId":"m-3","originalTimestamp":"2026-10-01T17:55:02.000Z","properties":{"plan":"pro","price":49.99,"orderId":9007199254740993},"context":{"library":{"name":"RudderLabs JavaScript SDK"}}},
{"type":"track","event":"Order Completed","userId":"user-1042","anonymousId":"anon-7f3a","messageId":"m-4","originalTimestamp":"2026-10-01T17:55:03.000Z","properties":{"revenue":49.99,"currency":"USD","products":[{"sku":"P-1","name":"Pro plan"}]},"context":{"library":{"name":"RudderLabs JavaScript SDK"}}},
{"type":"track","event":"<img src=x onerror=alert(1)>","anonymousId":"anon-9","messageId":"m-5","originalTimestamp":"2026-10-01T17:55:04.000Z","properties":{"note":"x'; rm -rf ~; echo"}}
]}`);
  await listener.send('/v1/track', {
    type: 'track', event: 'Signup Started', anonymousId: 'anon-22', messageId: 'm-6',
    originalTimestamp: '2026-10-01T17:55:05.000Z', properties: { source: 'ad' }, context: ctx,
  }, 'web-key-0123456789');
  await listener.send('/v1/track', { type: 'track', event: 'Order Completed', messageId: 'm-bad', properties: { revenue: 1 } });
  await listener.send('/v1/batch', '{"batch":[{"type":"track",');
  await fetch(`${listener.url}/sourceConfig/?p=npm&v=3.11.0&writeKey=dev`);
}

const test = base.extend({
  listener: async ({}, use) => {
    const listener = await startListener();
    await use(listener);
    await listener.stop();
  },
  // Every test fails on a console error. A test that makes the listener
  // refuse the page on purpose clears the network lines it expects. It
  // depends on listener, so it checks before the listener stops.
  consoleErrors: [async ({ page, listener }, use) => {
    const errors = [];
    page.on('console', (msg) => { if (msg.type() === 'error') errors.push(msg.text()); });
    await use(errors);
    expect(errors, 'console errors').toEqual([]);
  }, { auto: true }],
  // Every test fails on an uncaught error or an unhandled rejection.
  pageErrors: [async ({ page }, use) => {
    const errors = [];
    page.on('pageerror', (err) => errors.push(err.message));
    await use(errors);
    expect(errors, 'uncaught errors in the page').toEqual([]);
  }, { auto: true }],
});

// open loads the page and waits for the first whole load.
async function open(page, listener, query = '') {
  await page.goto(listener.ui + query);
  await expect(page.locator('#live-text')).toHaveText('Live');
  await expect(page.locator('#tiles .tile')).toHaveCount(6);
}

// apiRequests records the query of each request the page sends to one API
// route.
function apiRequests(page, route) {
  const seen = [];
  page.on('request', (req) => {
    const url = new URL(req.url());
    if (url.pathname === '/_dev/v1/' + route) seen.push(url.searchParams);
  });
  return seen;
}

// lastList returns the newest list request. The summary and the long-poll
// carry no filters and can follow it in any order.
function lastList(seen) {
  return seen.filter((q) => q.get('view') !== 'counts').at(-1);
}

module.exports = { test, expect, seed, open, apiRequests, lastList };
