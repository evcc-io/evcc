import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { exec, execSync } from "child_process";
import playwrightConfig from "../playwright.config";

const BASE_URL = playwrightConfig.use.baseURL;

const DB_PATH = "./evcc.db";
const BINARY = "./evcc";

export async function start(config, database) {
  clean();
  if (database) {
    console.log("loading database", { database });
    execSync(`sqlite3 ${DB_PATH} < tests/${database}`);
  }
  console.log("starting evcc", { config });
  const instance = exec(`EVCC_DATABASE_DSN=${DB_PATH} ${BINARY} --config tests/${config}`);
  instance.stdout.pipe(process.stdout);
  instance.on("exit", (code) => {
    if (code !== 0) {
      throw new Error("evcc terminated", code);
    }
  });
  await waitOn({ resources: [BASE_URL] });
}

export async function stop() {
  console.log("shutting down evcc");
  await axios.post(BASE_URL + "/api/shutdown");
  clean();
}

function clean() {
  if (fs.existsSync(DB_PATH)) {
    console.log("delete database", DB_PATH);
    fs.unlinkSync(DB_PATH);
  }
}
