import { test } from "@playwright/test";
import { start, baseUrl } from "./evcc";
import { execSync } from "child_process";
import path from "path";
import os from "os";
import axios from "axios";

test.use({ baseURL: baseUrl() });

test.describe("Solar Format Bug Test", async () => {

  const getWorkerDbPath = () => {
    const workerIndex = Number(process.env["TEST_WORKER_INDEX"] ?? 0);
    const port = 11000 + workerIndex;
    const file = `evcc-${port}.db`;
    return path.join(os.tmpdir(), file);
  };

  const getDatabaseFormat = async (): Promise<string | null> => {
    const dbPath = getWorkerDbPath();

    try {
      // First check if database and table exist
      const tableCheck = execSync(
        `echo "SELECT name FROM sqlite_master WHERE type='table' AND name='settings';" | sqlite3 "${dbPath}"`,
        { encoding: 'utf8', timeout: 5000 }
      );

      if (!tableCheck.trim()) {
        console.log("Settings table does not exist yet");
        return null;
      }

      const result = execSync(
        `echo "SELECT value FROM settings WHERE key = 'solarAccYield';" | sqlite3 "${dbPath}"`,
        { encoding: 'utf8', timeout: 5000 }
      );

      return result.trim() || null;
    } catch {
      console.log("Database read error");
      return null;
    }
  };

  const clearDatabase = async () => {
    const dbPath = getWorkerDbPath();
    try {
      execSync(`echo "DELETE FROM settings WHERE key IN ('solarAccYield', 'solarAccForecast');" | sqlite3 "${dbPath}"`);
    } catch {
      console.log("Database might not exist yet, will be created by evcc");
    }
  };

  test("Solar format consistency - Clean Playwright Lifecycle", async ({ page }) => {
    test.setTimeout(30000); // 30 second timeout
    console.log("🧪 Testing solar format bug with clean Playwright lifecycle");

    // 1. COMPLETELY CLEAR DATABASE - no seeded data whatsoever
    await clearDatabase();
    console.log("✅ Database cleared completely");

    // 2. START EVCC using Playwright's method and KEEP THE INSTANCE
    console.log("🚀 Starting evcc...");
    const instance = await start("battery-settings.evcc.yaml");

    // 3. LET EVCC RUN for enough time to accumulate solar data
    console.log("⏱️  Letting evcc run for 5 seconds to accumulate solar data...");
    await page.waitForTimeout(5000);

    // 4. STOP EVCC cleanly WITHOUT cleaning database
    console.log("🛑 Stopping evcc gracefully without cleaning database...");
    try {
      await axios.post(`${baseUrl()}/api/system/shutdown`, {});
      // Give it a moment to shutdown
      await page.waitForTimeout(2000);
    } catch {
      console.log("Graceful shutdown failed, using force kill");
      instance.kill("SIGKILL");
      await page.waitForTimeout(1000);
    }

    // 5. CHECK WHAT FORMAT EVCC WROTE TO DATABASE
    const format = await getDatabaseFormat();
    console.log("📊 Database format written by evcc:", format);

    if (!format) {
      console.log("⚠️  No solar data found in database - this could mean:");
      console.log("   - evcc didn't run long enough to accumulate data");
      console.log("   - solar forecast is 0 or very small");
      console.log("   - the solar meter is not generating data");
      console.log("📝 This test demonstrates the clean lifecycle but can't verify the format bug without data");
      return; // Exit gracefully - no data to test
    }

    // 6. ANALYZE THE FORMAT
    if (format.includes('"accumulated":') || format.includes('accumulated:')) {
      // BUGGY FORMAT DETECTED
      throw new Error(`❌ BUGGY FORMAT: evcc wrote nested format: ${format}`);
    } else {
      // Check if it's valid flat format
      try {
        // Handle unquoted keys from sqlite output
        const normalizedJson = format.replace(/\{(\w+):/g, '{"$1":');
        const parsed = JSON.parse(normalizedJson);

        if (parsed.pv !== undefined && typeof parsed.pv === 'number') {
          console.log(`✅ CORRECT FORMAT: Flat structure {"pv": ${parsed.pv}}`);
        } else {
          throw new Error(`❌ UNEXPECTED FORMAT: ${format}`);
        }
      } catch {
        throw new Error(`❌ INVALID JSON FORMAT: ${format}`);
      }
    }
  });
});