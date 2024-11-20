import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { spawn, execSync } from "child_process";
import os from "os";
import path from "path";
import { Transform } from "stream";

const BINARY = "./evcc";

function workerPort() {
  const index = process.env.TEST_WORKER_INDEX * 1;
  return 11000 + index;
}

function logPrefix() {
  return `[worker:${process.env.TEST_WORKER_INDEX}]`;
}

function createSteamLog() {
  return new Transform({
    transform(chunk, encoding, callback) {
      const lines = chunk.toString().split("\n");
      lines.forEach((line) => {
        if (line.trim()) log(line);
      });
      callback();
    },
  });
}

function log(...args) {
  console.log(logPrefix(), ...args);
}

export function baseUrl() {
  return `http://localhost:${workerPort()}`;
}

function dbPath() {
  const file = `evcc-${workerPort()}.db`;
  return path.join(os.tmpdir(), file);
}

export async function start(config, sqlDumps, flags = "--disable-auth") {
  await _clean();
  if (sqlDumps) {
    await _restoreDatabase(sqlDumps);
  }
  return await _start(config, flags);
}

export async function stop(instance) {
  await _stop(instance);
  await _clean();
}

export async function restart(config, flags = "--disable-auth") {
  await _stop();
  await _start(config, flags);
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
    log("loading database", dbPath(), dump);
    execSync(`sqlite3 ${dbPath()} < tests/${dump}`);
  }
}

async function _start(config, flags = []) {
  const configFile = config.includes("/") ? config : `tests/${config}`;
  const port = workerPort();
  log(`wait until port ${port} is available`);
  // wait for port to be available
  await waitOn({ resources: [`tcp:${port}`], reverse: true, log: true });
  const additionalFlags = typeof flags === "string" ? [flags] : flags;
  log("starting evcc", { config, port, additionalFlags });
  const instance = spawn(BINARY, ["--config", configFile, additionalFlags], {
    env: { EVCC_NETWORK_PORT: port.toString(), EVCC_DATABASE_DSN: dbPath() },
    stdio: ["pipe", "pipe", "pipe"],
  });
  const steamLog = createSteamLog();
  instance.stdout.pipe(steamLog);
  instance.stderr.pipe(steamLog);
  instance.on("exit", (code) => {
    log("evcc terminated", { code, port, config });
    steamLog.end();
  });
  await waitOn({ resources: [baseUrl()], log: true });
  return instance;
}

async function _stop(instance) {
  const port = workerPort();
  if (instance) {
    log("shutting down evcc hard", { port });
    instance.kill("SIGKILL");
    await waitOn({ resources: [`tcp:${port}`], reverse: true, log: true });
    log("evcc is down", { port });
    return;
  }
  // check if auth is required
  const res = await axios.get(`${baseUrl()}/api/auth/status`);
  log("auth status", res.status, res.statusText, res.data);
  let cookie;
  // login required
  if (!res.data) {
    const res = await axios.post(`${baseUrl()}/api/auth/login`, { password: "secret" });
    log("login", res.status, res.statusText);
    cookie = res.headers["set-cookie"];
  }
  log("shutting down evcc", { port });
  await axios.post(`${baseUrl()}/api/system/shutdown`, {}, { headers: { cookie } });
  log(`wait until port ${port} is closed`);
  await waitOn({ resources: [`tcp:${port}`], reverse: true, log: true });
  log("evcc is down", { port });
}

async function _clean() {
  const db = dbPath();
  if (fs.existsSync(db)) {
    log("delete database", db);
    fs.unlinkSync(db);
  }
}
