// Opens the fixture page in Chromium, clicks the three buttons and waits for
// the three /v1/track responses. Usage: node fire.mjs PAGE_URL
import { chromium } from 'playwright';

const pageUrl = process.argv[2];
if (!pageUrl) {
  console.error('usage: node fire.mjs PAGE_URL');
  process.exit(2);
}

const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  let tracked = 0;
  const allTracked = new Promise((resolve) =>
    page.on('response', (r) => {
      if (new URL(r.url()).pathname === '/v1/track' && r.request().method() === 'POST' && ++tracked === 3) resolve();
    }),
  );
  await page.goto(pageUrl);
  for (const id of ['shown', 'clicked', 'sent']) await page.click(`#${id}`);
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(`${tracked} of 3 /v1/track responses in 20s`)), 20_000);
  });
  await Promise.race([allTracked, timeout]).finally(() => clearTimeout(timer));
} finally {
  await browser.close();
}
console.log('fired');
