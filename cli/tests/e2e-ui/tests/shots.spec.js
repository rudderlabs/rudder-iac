// Review screenshots of the page. Runs only when SHOTS_DIR names a folder.
const path = require('node:path');
const { test, expect, seed, open } = require('./fixtures');

const dir = process.env.SHOTS_DIR;

test.describe('screenshots', () => {
  test.skip(!dir, 'set SHOTS_DIR to take the screenshots');

  const shot = (page, name) => page.screenshot({ path: path.join(dir, name), fullPage: false });

  for (const [scheme, width, shots] of [
    ['dark', 1280, ['01-events-1280-dark.png']],
    ['light', 1280, ['02-event-detail-1280-light.png', '03-refused-request-1280-light.png', '04-filter-live-1280-light.png']],
    ['dark', 400, ['05-events-400-dark.png', '06-event-detail-400-dark.png']],
  ]) {
    test(`${scheme} ${width}`, async ({ page, listener, consoleErrors }) => {
      await page.emulateMedia({ colorScheme: scheme });
      await page.setViewportSize({ width, height: width > 500 ? 900 : 860 });
      await seed(listener);
      await open(page, listener);
      await expect(page.locator('#count')).toHaveText('6 accepted events');
      const rows = page.locator('#list-body tr:not([hidden])');

      for (const name of shots) {
        if (name.startsWith('02') || name.startsWith('06')) {
          await rows.filter({ hasText: 'Plan Selected' }).click();
          await expect(page.locator('#detail')).toContainText('Open this request');
        }
        if (name.startsWith('03')) {
          await page.getByRole('tab', { name: 'Requests' }).click();
          await page.locator('#list-body tr', { hasText: '400' }).first().click();
          await expect(page.locator('#detail .reason')).toBeVisible();
        }
        if (name.startsWith('04')) {
          await page.getByRole('tab', { name: 'Events' }).click();
          await page.getByLabel('Event name', { exact: true }).fill('Order*');
          await page.getByLabel('Event name', { exact: true }).press('Tab');
          await expect(page.locator('#count')).toHaveText('Showing 1 of 6 accepted events');
          await listener.send('/v1/track', { type: 'track', event: 'Order Refunded', userId: 'user-1042', messageId: 'm-7' });
          await expect(page.locator('#list-body tr.fresh')).toHaveCount(1);
        }
        await shot(page, name);
      }
      expect(consoleErrors, 'console errors').toEqual([]);
    });
  }
});
