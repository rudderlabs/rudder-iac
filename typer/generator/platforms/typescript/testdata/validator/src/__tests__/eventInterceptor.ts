import { http, HttpResponse } from "msw";
import { setupServer, type SetupServer } from "msw/node";

export const TEST_DATA_PLANE_URL = "https://dataplane.test.invalid";
export const TEST_CONFIG_BE_URL = "https://config.test.invalid";
export const TEST_WRITE_KEY = "test-write-key";

export type CapturedEvent = {
  type: string;
  userId?: string | null;
  anonymousId?: string;
  event?: string;
  groupId?: string | null;
  properties?: Record<string, unknown>;
  traits?: Record<string, unknown>;
  context?: Record<string, unknown>;
  [key: string]: unknown;
};

const SOURCE_CONFIG_RESPONSE = {
  source: {
    id: "test-source",
    name: "test-source",
    writeKey: TEST_WRITE_KEY,
    enabled: true,
    workspaceId: "test-workspace",
    destinations: [],
    config: {
      statsCollection: {
        errors: { enabled: false },
        metrics: { enabled: false },
      },
    },
  },
};

export class EventInterceptor {
  private readonly server: SetupServer;
  private readonly received: CapturedEvent[] = [];

  constructor() {
    this.server = setupServer(
      http.get(`${TEST_CONFIG_BE_URL}/sourceConfig/`, () =>
        HttpResponse.json(SOURCE_CONFIG_RESPONSE),
      ),
      http.post(`${TEST_DATA_PLANE_URL}/v1/:eventType`, async ({ request }) => {
        this.received.push((await request.json()) as CapturedEvent);
        return HttpResponse.text("OK", { status: 200 });
      }),
    );
  }

  start(): void {
    this.server.listen({ onUnhandledRequest: "error" });
  }

  stop(): void {
    this.server.close();
  }

  reset(): void {
    this.received.length = 0;
    this.server.resetHandlers();
  }

  events(): readonly CapturedEvent[] {
    return this.received;
  }

  async waitForEvents(count: number, timeoutMs = 5_000): Promise<CapturedEvent[]> {
    const deadline = Date.now() + timeoutMs;
    while (this.received.length < count) {
      if (Date.now() > deadline) {
        throw new Error(
          `Timed out waiting for ${count} event(s); received ${this.received.length} after ${timeoutMs}ms`,
        );
      }
      await new Promise((r) => setTimeout(r, 10));
    }
    return this.received.slice(0, count);
  }
}

export const interceptor = new EventInterceptor();
