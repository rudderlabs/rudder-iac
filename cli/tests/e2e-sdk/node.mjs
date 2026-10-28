// The Node SDK with gzip on: it posts a gzip body to /v1/batch.
// Usage: node node.mjs DATA_PLANE_URL
import Analytics from '@rudderstack/rudder-sdk-node';

const dataPlaneUrl = process.argv[2];
if (!dataPlaneUrl) {
  console.error('usage: node node.mjs DATA_PLANE_URL');
  process.exit(2);
}
const client = new Analytics('node', { dataPlaneUrl, gzip: true });
client.track({ userId: 'u1', event: 'SDK Probe', properties: { sdk: 'node', n: 1 } });
await client.flush();
console.log('node sent');
