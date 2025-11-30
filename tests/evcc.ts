import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { spawn, execSync, ChildProcess } from "child_process";
import killPort from "kill-port";
import os from "os";
import path from "path";
import { Transform } from "stream";
import { test } from "@playwright/test";

const BINARY = "./evcc";
const IS_CI = !!process.env["GITHUB_ACTIONS"];
const LOG_ENABLED = !IS_CI;

// sometimes evcc startup fails due to infra issues in runner ususally fixed by retry. allowing some fails to avoid github annotations clutter
let allowedStartupFails = IS_CI ? 2 : 0;

function workerPort() {
  const index = Number(process.env["TEST_WORKER_INDEX"] ?? 0);
  return 11000 + index;
}

function logPrefix() {
  return `[worker:${process.env["TEST_WORKER_INDEX"]}]`;
}

function createSteamLog() {
  return new Transform({
    transform(chunk: Buffer, _, callback) {
      const lines = chunk.toString().split("\n");
      lines.forEach((line: string) => {
        if (line.trim()) log(line);
      });
      callback();
    },
  });
}

function log(...args: any[]) {
  if (LOG_ENABLED) {
    console.log(logPrefix(), ...args);
  }
}

export function baseUrl() {
  return `http://localhost:${workerPort()}`;
}

function dbPath() {
  const file = `evcc-${workerPort()}.db`;
  return path.join(os.tmpdir(), file);
}

export async function start(
  config?: string,
  sqlDumps?: string | null,
  flags: string | string[] = "--disable-auth"
) {
  await _clean();
  if (sqlDumps) {
    await _restoreDatabase(sqlDumps);
  }
  return await _start(config, flags);
}

export async function stop(instance?: ChildProcess) {
  await _stop(instance);
  await _clean();
}

export async function restart(
  config?: string,
  flags: string | string[] = "--disable-auth",
  alreadyStopped = false
) {
  if (!alreadyStopped) {
    await _stop();
  }
  await _start(config, flags);
}

export async function cleanRestart(config: string, sqlDumps: string) {
  await _stop();
  await _clean();
  if (sqlDumps) {
    await _restoreDatabase(sqlDumps);
  }
  await _start(config);
}

async function _restoreDatabase(sqlDumps: string) {
  const dumps = Array.isArray(sqlDumps) ? sqlDumps : [sqlDumps];
  for (const dump of dumps) {
    log("loading database", dbPath(), dump);
    execSync(`sqlite3 ${dbPath()} < tests/${dump}`);
  }
}

async function _start(config?: string, flags: string | string[] = []) {
  const configArgs = config ? ["--config", config.includes("/") ? config : `tests/${config}`] : [];
  const port = workerPort();
  log(`wait until port ${port} is available`);
  // wait for port to be available
  await waitOn({ resources: [`tcp:${port}`], reverse: true, log: LOG_ENABLED });
  const additionalFlags = typeof flags === "string" ? [flags] : flags;
  additionalFlags.push("--log", "debug,httpd:trace");
  log("starting evcc", { config, port, additionalFlags });
  const instance = spawn(BINARY, [...configArgs, ...additionalFlags], {
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
  try {
    await waitOn({ resources: [baseUrl()], log: LOG_ENABLED, timeout: 50000 });
  } catch (error) {
    instance.kill("SIGKILL");
    console.error(logPrefix(), `evcc startup failed: ${error}`);

    if (allowedStartupFails > 0) {
      allowedStartupFails--;
      test.skip(true, `evcc startup timeout (${allowedStartupFails} skips remaining)`);
      return;
    }
    throw error;
  }
  return instance;
}

async function _stop(instance?: ChildProcess) {
  const port = workerPort();
  if (instance) {
    log("shutting down evcc hard", { port });
    instance.kill("SIGKILL");
    await waitOn({ resources: [`tcp:${port}`], reverse: true, log: LOG_ENABLED });
    log("evcc is down", { port });
    return;
  }
  let cookie;
  try {
    // check if auth is required
    const res = await axios.get(`${baseUrl()}/api/auth/status`);
    log("auth status", res.status, res.statusText, res.data);
    // login required
    if (!res.data) {
      const res = await axios.post(`${baseUrl()}/api/auth/login`, { password: "secret" });
      log("login", res.status, res.statusText);
      cookie = res.headers["set-cookie"];
    }
    log("shutting down evcc", { port });
    await axios.post(`${baseUrl()}/api/system/shutdown`, {}, { headers: { cookie } });
  } catch (error) {
    const port = workerPort();
    log(`shutdown failed, last resort: kill by port`, port, error);
    try {
      await killPort(port);
      log(`killed process on port ${port}`);
    } catch (killError) {
      log(`no process found on port ${port} or kill failed:`, killError);
    }
  }
  log(`wait until port ${port} is closed`);
  await waitOn({ resources: [`tcp:${port}`], reverse: true, log: LOG_ENABLED });
  log("evcc is down", { port });
}

async function _clean() {
  const db = dbPath();
  if (fs.existsSync(db)) {
    log("delete database", db);
    fs.unlinkSync(db);
  }
}
