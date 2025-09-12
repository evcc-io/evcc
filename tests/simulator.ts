import os from "os";
import path from "path";
import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { spawn } from "child_process";
import { Transform } from "stream";
import type { Page } from "@playwright/test";

const LOG_ENABLED = false;

function workerPort() {
  const index = parseInt(process.env["TEST_WORKER_INDEX"] ?? "-1");
  return 12000 + index;
}

function logPrefix() {
  return `[worker:${process.env["TEST_WORKER_INDEX"]}]`;
}

function createSteamLog() {
  return new Transform({
    transform(chunk: Buffer, _, callback) {
      const lines = chunk.toString().split("\n");
      lines.forEach((line) => {
        if (line.trim()) log(line);
      });
      callback();
    },
  });
}

function log(...args: any[]) {
  // uncomment for debugging
  if (LOG_ENABLED) {
    console.log(logPrefix(), ...args);
  }
}

export function simulatorHost() {
  return `localhost:${workerPort()}`;
}

export function simulatorUrl() {
  return `http://${simulatorHost()}`;
}

export function simulatorConfig() {
  const input = "./tests/simulator.evcc.yaml";
  const content = fs.readFileSync(input, "utf8");
  const result = content.replace(/localhost:7072/g, simulatorHost());
  const resultName = "simulator.evcc.generated.yaml";
  const resultPath = path.join(os.tmpdir(), resultName);
  fs.writeFileSync(resultPath, result);
  return resultPath;
}

export async function startSimulator() {
  const port = workerPort();
  log("starting simulator", { port });
  log(`wait until port ${port} is available`);
  await waitOn({ resources: [`tcp:${port}`], reverse: true, log: LOG_ENABLED });

  const instance = spawn("npm", ["run", "simulator", "--", "--port", port.toString()]);

  const steamLog = createSteamLog();
  instance.stdout.pipe(steamLog);
  instance.stderr.pipe(steamLog);
  instance.on("exit", (code) => {
    log("simulator terminated", { code, port });
    steamLog.end();
  });

  await waitOn({ resources: [`${simulatorUrl()}/api/state`], log: LOG_ENABLED });
}

export async function stopSimulator() {
  const port = workerPort();
  log("shutting down simulator", { port });
  await axios.post(`${simulatorUrl()}/api/shutdown`);
  log(`wait until port ${port} is closed`);
  await waitOn({ resources: [`tcp:localhost:${port}`], reverse: true, log: LOG_ENABLED });
}

export const simulatorApply = async (page: Page) => {
  await Promise.all([
    page.waitForResponse(
      (response) => response.url().includes("/api/state") && response.request().method() === "POST"
    ),
    page.getByRole("button", { name: "Apply changes" }).click(),
  ]);
};
