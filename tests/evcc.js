import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { exec, execSync } from "child_process";
import playwrightConfig from "../playwright.config.js";

const BASE_URL = playwrightConfig.use.baseURL;

const DB_PATH = "./evcc.db";
const BINARY = "./evcc";

export async function start(config, sqlDumps) {
  await _clean();
  if (sqlDumps) {
    await _restoreDatabase(sqlDumps);
  }
  await _start(config);
}

export async function stop() {
  await _stop();
  await _clean();
}

export async function restart(config) {
  await _stop();
  await _start(config);
}

export async function cleanRestart(config, sqlDumps) {
  await _stop();
  await _clean();
  if (sqlDumps) {
    await _restoreDatabase(sqlDumps);
  }
  await _start(config);
}

async function _restoreDatabase(sqlDumps) {
  const dumps = Array.isArray(sqlDumps) ? sqlDumps : [sqlDumps];
  for (const dump of dumps) {
    console.log("loading database", dump);
    execSync(`sqlite3 ${DB_PATH} < tests/${dump}`);
  }
}

async function _start(config) {
  console.log("starting evcc", { config });
  const instance = exec(`EVCC_DATABASE_DSN=${DB_PATH} ${BINARY} --config tests/${config}`);
  instance.stdout.pipe(process.stdout);
  instance.stderr.pipe(process.stderr);
  instance.on("exit", (code) => {
    if (code !== 0) {
      throw new Error("evcc terminated", code);
    }
  });
  await waitOn({ resources: [BASE_URL] });
}

async function _stop() {
  console.log("shutting down evcc");
  await axios.post(BASE_URL + "/api/shutdown");
}

async function _clean() {
  if (fs.existsSync(DB_PATH)) {
    console.log("delete database", DB_PATH);
    fs.unlinkSync(DB_PATH);
  }
}
