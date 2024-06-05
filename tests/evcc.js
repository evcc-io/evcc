import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { exec, execSync } from "child_process";
import os from "os";
import path from "path";

const BINARY = "./evcc";

function port() {
  const index = process.env.TEST_WORKER_INDEX * 1;
  return 11000 + index;
}

export function baseUrl() {
  return `http://localhost:${port()}`;
}

function dbPath() {
  const file = `evcc-${port()}.db`;
  return path.join(os.tmpdir(), file);
}

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
    console.log("loading database", dbPath(), dump);
    execSync(`sqlite3 ${dbPath()} < tests/${dump}`);
  }
}

async function _start(config) {
  const configFile = config.includes("/") ? config : `tests/${config}`;
  console.log("starting evcc", { config });
  const instance = exec(
    `EVCC_NETWORK_PORT=${port()} EVCC_DATABASE_DSN=${dbPath()} ${BINARY} --config ${configFile}`
  );
  instance.stdout.pipe(process.stdout);
  instance.stderr.pipe(process.stderr);
  instance.on("exit", (code) => {
    console.log("evcc terminated", code);
  });
  await waitOn({ resources: [baseUrl()] });
}

async function _stop() {
  console.log("shutting down evcc");
  const res = await axios.post(`${baseUrl()}/api/auth/login`, { password: "secret" });
  console.log(res.status, res.statusText);
  const cookie = res.headers["set-cookie"];
  await axios.post(`${baseUrl()}/api/system/shutdown`, {}, { headers: { cookie } });
  console.log("wait until network port is closed");
  await waitOn({ resources: [`tcp:localhost:${port()}`], reverse: true });
  console.log("evcc is down");
}

async function _clean() {
  const db = dbPath();
  if (fs.existsSync(db)) {
    console.log("delete database", db);
    fs.unlinkSync(db);
  }
}
