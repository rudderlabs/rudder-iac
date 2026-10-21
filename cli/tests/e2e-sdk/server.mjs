// Serves the fixture page and the browser SDK on 127.0.0.1:0.
// Run directly, it reads WRITE_KEY, DATA_PLANE_URL and DEFECT from env and
// prints {"url": ...} on stdout.
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SDK = path.join(HERE, 'node_modules/@rudderstack/analytics-js/dist/npm/modern/bundled/esm/index.mjs');
const DEFECTS = ['none', 'wrong-type'];

// probe: the page tracks one 'SDK Probe' when the SDK is ready, instead of
// waiting for button clicks.
export function start({ writeKey, dataPlaneUrl, defect = 'none', probe = false }) {
  if (!writeKey || !dataPlaneUrl) throw new Error('writeKey and dataPlaneUrl are required');
  if (!DEFECTS.includes(defect)) throw new Error(`defect must be one of ${DEFECTS.join(', ')}`);
  // JSON inside a <script>: escape < so a value cannot close the tag.
  const config = JSON.stringify({ writeKey, dataPlaneUrl, defect, probe }).replace(/</g, '\\u003c');
  const page = fs.readFileSync(path.join(HERE, 'index.html'), 'utf8').replace('__CONFIG__', config);

  const server = http.createServer((req, res) => {
    const { pathname } = new URL(req.url, 'http://x');
    if (pathname === '/') {
      res.writeHead(200, { 'content-type': 'text/html; charset=utf-8', 'cache-control': 'no-store' });
      res.end(page);
    } else if (pathname === '/sdk.mjs') {
      res.writeHead(200, { 'content-type': 'text/javascript; charset=utf-8' });
      fs.createReadStream(SDK).pipe(res);
    } else {
      res.writeHead(404).end('not found');
    }
  });
  return new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () =>
      resolve({ url: `http://127.0.0.1:${server.address().port}/`, close: () => server.close() }),
    );
  });
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const { url } = await start({
    writeKey: process.env.WRITE_KEY,
    dataPlaneUrl: process.env.DATA_PLANE_URL,
    defect: process.env.DEFECT ?? 'none',
  });
  console.log(JSON.stringify({ url }));
  for (const sig of ['SIGINT', 'SIGTERM']) process.on(sig, () => process.exit(0));
}
