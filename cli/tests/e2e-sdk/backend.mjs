// The fixture app's server side: one track, then flush.
// Usage: node backend.mjs DATA_PLANE_URL
import Analytics from '@rudderstack/rudder-sdk-node';

const dataPlaneUrl = process.argv[2];
if (!dataPlaneUrl) {
  console.error('usage: node backend.mjs DATA_PLANE_URL');
  process.exit(2);
}
const client = new Analytics('api', { dataPlaneUrl });
client.track({ userId: 'u1', event: 'Order Completed', properties: { orderId: 'o1', total: 99.5 } });
await client.flush();
console.log('sent');
