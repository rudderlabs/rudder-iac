import { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { TEST_CONFIG_BE_URL, TEST_DATA_PLANE_URL, TEST_WRITE_KEY } from "./eventInterceptor.ts";

// The SDK persists user state in localStorage and cookies, so state leaks
// between tests in the same file unless it is cleared. Trait persistence is
// exactly what these tests measure, so a leaked trait reads as a pass.
export async function freshAnalytics(): Promise<RudderAnalytics> {
  localStorage.clear();
  sessionStorage.clear();
  for (const c of document.cookie.split(";")) {
    document.cookie = `${c.replace(/^ +/, "").replace(/=.*/, "")}=;expires=${new Date(0).toUTCString()};path=/`;
  }

  const analytics = new RudderAnalytics();
  analytics.load(TEST_WRITE_KEY, TEST_DATA_PLANE_URL, {
    configUrl: TEST_CONFIG_BE_URL,
    logLevel: "ERROR",
    queueOptions: { maxItems: 100, batch: { enabled: false } },
    sessions: { autoTrack: false },
    uaChTrackLevel: "none",
  });
  await new Promise<void>((resolve) => analytics.ready(() => resolve()));
  (analytics as unknown as { reset: (b: boolean) => void }).reset(true);
  return analytics;
}

export const traitsOf = (event: Record<string, unknown>): unknown =>
  (event.context as Record<string, unknown> | undefined)?.traits;
