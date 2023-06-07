import fs from "fs";
import waitOn from "wait-on";
import { exec, execSync } from "child_process";
import playwrightConfig from "../playwright.config";

const BASE_URL = playwrightConfig.use.baseURL;

const DB_PATH = "./evcc.db";
const BINARY = "./evcc";

let instance = null;

export async function start(config, database) {
  await stop();
  if (database) {
    console.log("loading database", { database });
    execSync(`sqlite3 ${DB_PATH} < tests/${database}`);
  }
  console.log("starting evcc", { config });
  instance = exec(`EVCC_DATABASE_DSN=${DB_PATH} ${BINARY} --config tests/${config}`);
  instance.stdout.pipe(process.stdout);
  instance.on("exit", (code) => {
    if (code !== 0) {
      throw new Error("evcc terminated", code);
    }
  });
  await waitOn({ resources: [BASE_URL] });
}

export async function stop() {
  if (!instance) return;
  const result = new Promise((resolve) => instance.on("exit", resolve));
  instance.on("exit", () => {
    if (fs.existsSync(DB_PATH)) {
      console.log("delete database", DB_PATH);
      fs.unlinkSync(DB_PATH);
    }
  });
  console.log("stopping evcc");
  instance.kill();

  instance = null;
  return result;
}
