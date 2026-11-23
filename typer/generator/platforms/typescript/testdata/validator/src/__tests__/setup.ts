import { afterAll, afterEach, beforeAll } from "vitest";
import { interceptor } from "./eventInterceptor.ts";

beforeAll(() => interceptor.start());
afterEach(() => interceptor.reset());
afterAll(() => interceptor.stop());
