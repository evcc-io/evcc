import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { validateState } from "../scripts/state-schema/validate";

test.use({ baseURL: baseUrl() });

const CONFIG = "cmd/demo.yaml";

test.beforeAll(async () => {
  await start(CONFIG);
});
test.afterAll(async () => {
  await stop();
});

test("/api/state matches the openapi State schema", async ({ request }) => {
  // loadpoint values appear over the first update cycles, poll until stable
  await expect
    .poll(
      async () => {
        const res = await request.get("/api/state");
        const { errors, undocumented } = validateState(await res.json());
        return [...errors, ...undocumented];
      },
      { timeout: 30000 }
    )
    .toEqual([]);
});
