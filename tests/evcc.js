import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { exec, execSync } from "child_process";
import os from "os";
import path from "path";
import { Transform } from "stream";

const BINARY = "./evcc";

const waitOpts = {
  timeout: 20000,
  log: true,
};

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
    log("loading database", dbPath(), dump);
    execSync(`sqlite3 ${dbPath()} < tests/${dump}`);
  }
}

async function _start(config) {
  const configFile = config.includes("/") ? config : `tests/${config}`;
  const port = workerPort();
  log(`wait until port ${port} is available`);
  // wait for port to be available
  await waitOn({ resources: [`tcp:${port}`], reverse: true, ...waitOpts });
  log("starting evcc", { config, port });
  const instance = exec(
    `EVCC_NETWORK_PORT=${port} EVCC_DATABASE_DSN=${dbPath()} ${BINARY} --config ${configFile}`
  );
  const steamLog = createSteamLog();
  instance.stdout.pipe(steamLog);
  instance.stderr.pipe(steamLog);
  instance.on("exit", (code) => {
    log("evcc terminated", { code, port, config });
    steamLog.end();
  });
  await waitOn({ resources: [baseUrl()], ...waitOpts });
  return instance;
}

async function _stop(instance) {
  const port = workerPort();
  if (instance) {
    log("shutting down evcc hard", { port });
    // hard kill, only use of normal shutdown doesn't work
    instance.kill("SIGKILL");
    await waitOn({ resources: [`tcp:${port}`], reverse: true, ...waitOpts });

    log("evcc is down", { port });
    return;
  }
  log("shutting down evcc", { port });
  const res = await axios.post(`${baseUrl()}/api/auth/login`, { password: "secret" });
  log(res.status, res.statusText);
  const cookie = res.headers["set-cookie"];
  await axios.post(`${baseUrl()}/api/system/shutdown`, {}, { headers: { cookie } });
  log(`wait until port ${port} is closed`);
  await waitOn({ resources: [`tcp:${port}`], reverse: true, ...waitOpts });
  log("evcc is down", { port });
}

async function _clean() {
  const db = dbPath();
  if (fs.existsSync(db)) {
    log("delete database", db);
    fs.unlinkSync(db);
  }
}
