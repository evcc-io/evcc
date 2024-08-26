import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { exec, execSync } from "child_process";
import os from "os";
import path from "path";

const BINARY = "./evcc";

function workerPort() {
  const index = process.env.TEST_WORKER_INDEX * 1;
  return 11000 + index;
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function baseUrl() {
  return `http://localhost:${workerPort()}`;
}

function dbPath() {
  const file = `evcc-${workerPort()}.db`;
  return path.join(os.tmpdir(), file);
}

export async function start(config, sqlDumps) {
  await _clean();
  if (sqlDumps) {
    await _restoreDatabase(sqlDumps);
  }
  return await _start(config);
}

export async function stop(instance) {
  await _stop(instance);
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
  const port = workerPort();
  console.log(`wait until port ${port} is available`);
  await waitOn({ resources: [`tcp:localhost:${port}`], reverse: true });
  console.log("starting evcc", { config, port });
  const instance = exec(
    `EVCC_NETWORK_PORT=${port} EVCC_DATABASE_DSN=${dbPath()} ${BINARY} --config ${configFile}`
  );
  instance.stdout.pipe(process.stdout);
  instance.stderr.pipe(process.stderr);
  instance.on("exit", (code) => {
    console.log("evcc terminated", { code, port, config });
  });
  await waitOn({ resources: [baseUrl()] });
  return instance;
}

async function _stop(instance) {
  if (instance) {
    console.log("shutting down evcc hard");
    // hard kill, only use of normal shutdown doesn't work
    instance.kill("SIGKILL");
    await sleep(300);
    return;
  }
  const port = workerPort();
  console.log("shutting down evcc", { port });
  const res = await axios.post(`${baseUrl()}/api/auth/login`, { password: "secret" });
  console.log(res.status, res.statusText);
  const cookie = res.headers["set-cookie"];
  await axios.post(`${baseUrl()}/api/system/shutdown`, {}, { headers: { cookie } });
  console.log(`wait until port ${port} is closed`);
  await waitOn({ resources: [`tcp:localhost:${port}`], reverse: true });
  console.log("evcc is down", { port });
}

async function _clean() {
  const db = dbPath();
  if (fs.existsSync(db)) {
    console.log("delete database", db);
    fs.unlinkSync(db);
  }
}
