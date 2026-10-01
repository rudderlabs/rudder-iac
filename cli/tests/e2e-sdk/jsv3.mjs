// The JS v3 SDK in Chromium, loaded from the fixture server with configUrl
// set to the listener. Usage: node jsv3.mjs DATA_PLANE_URL
import { chromium } from 'playwright';
import { start } from './server.mjs';

const dataPlaneUrl = process.argv[2];
if (!dataPlaneUrl) {
  console.error('usage: node jsv3.mjs DATA_PLANE_URL');
  process.exit(2);
}
const server = await start({ writeKey: 'jsv3', dataPlaneUrl, probe: true });
const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  const tracked = page.waitForResponse(
    (r) => new URL(r.url()).pathname === '/v1/track' && r.request().method() === 'POST',
    { timeout: 20_000 },
  );
  await page.goto(server.url);
  const resp = await tracked;
  if (resp.status() !== 200) throw new Error(`/v1/track answered ${resp.status()}`);
} finally {
  await browser.close();
  server.close();
}
console.log('jsv3 sent');
