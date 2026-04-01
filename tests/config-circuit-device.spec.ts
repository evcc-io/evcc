// Temporary API-level tests to validate circuit device CRUD endpoints.
// Will be replaced by UI-based tests once the circuit configuration UI is implemented.

import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeEach(async () => {
  await start();
});

test.afterEach(async () => {
  await stop();
});

test.describe("circuit device api", () => {
  test("crud lifecycle and restart persistence", async ({ page }) => {
    // create root circuit
    const createRoot = await page.request.post("/api/config/devices/circuit", {
      data: {
        type: "template",
        template: "static",
        title: "Main",
        maxcurrent: 32,
      },
    });
    expect(createRoot.status()).toBe(200);
    const root = await createRoot.json();
    expect(root).toHaveProperty("id");
    expect(root).toHaveProperty("name");

    // create child circuit with parent
    const createChild = await page.request.post("/api/config/devices/circuit", {
      data: {
        type: "template",
        template: "static",
        title: "Garage",
        maxcurrent: 16,
        parent: root.name,
      },
    });
    expect(createChild.status()).toBe(200);
    const child = await createChild.json();
    expect(child).toHaveProperty("id");

    // list circuits
    const list = await page.request.get("/api/config/devices/circuit");
    expect(list.status()).toBe(200);
    const listBody = await list.json();
    expect(listBody).toHaveLength(2);

    // get single
    const get = await page.request.get(`/api/config/devices/circuit/${root.id}`);
    expect(get.status()).toBe(200);
    const getBody = await get.json();
    expect(getBody).toHaveProperty("type", "template");
    expect(getBody).toHaveProperty("name", root.name);

    // update
    const update = await page.request.put(`/api/config/devices/circuit/${root.id}`, {
      data: {
        type: "template",
        template: "static",
        title: "Main Updated",
        maxcurrent: 48,
      },
    });
    expect(update.status()).toBe(200);

    // verify update
    const getUpdated = await page.request.get(`/api/config/devices/circuit/${root.id}`);
    const updatedBody = await getUpdated.json();
    expect(updatedBody.config.title).toBe("Main Updated");
    expect(updatedBody.config.maxcurrent).toBe(48);

    // delete child
    const del = await page.request.delete(`/api/config/devices/circuit/${child.id}`);
    expect(del.status()).toBe(200);

    // verify deleted
    const listAfter = await page.request.get("/api/config/devices/circuit");
    const listAfterBody = await listAfter.json();
    expect(listAfterBody).toHaveLength(1);

    // restart and verify persistence
    await restart();

    const listRestart = await page.request.get("/api/config/devices/circuit");
    expect(listRestart.status()).toBe(200);
    const restartBody = await listRestart.json();
    expect(restartBody).toHaveLength(1);
  });

  test("error handling", async ({ page }) => {
    // get non-existent circuit
    const get = await page.request.get("/api/config/devices/circuit/99");
    expect(get.status()).toBe(400);

    // delete non-existent circuit
    const del = await page.request.delete("/api/config/devices/circuit/99");
    expect(del.status()).toBe(400);

    // create with invalid template
    const invalid = await page.request.post("/api/config/devices/circuit", {
      data: {
        type: "template",
        template: "nonexistent",
        title: "Bad",
      },
    });
    expect(invalid.ok()).toBeFalsy();
  });

  test("custom type circuit", async ({ page }) => {
    // create circuit via custom type with raw YAML
    const create = await page.request.post("/api/config/devices/circuit", {
      data: {
        type: "custom",
        yaml: "type: custom\ntitle: Custom Circuit\nmaxCurrent: 24\n",
      },
    });
    expect(create.status()).toBe(200);
    const result = await create.json();
    expect(result).toHaveProperty("id");

    // verify it exists
    const list = await page.request.get("/api/config/devices/circuit");
    const body = await list.json();
    expect(body).toHaveLength(1);
  });
});
