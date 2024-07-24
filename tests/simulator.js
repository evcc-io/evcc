import os from "os";
import path from "path";
import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { exec } from "child_process";

function workerPort() {
  const index = process.env.TEST_WORKER_INDEX * 1;
  return 12000 + index;
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
  console.log("starting simulator", { port });
  console.log(`wait until port ${port} is available`);
  await waitOn({ resources: [`tcp:localhost:${port}`], reverse: true });
  const instance = exec(`npm run simulator -- --port ${port}`);
  instance.stdout.pipe(process.stdout);
  instance.stderr.pipe(process.stderr);

  instance.on("exit", (code) => {
    if (code !== 0) {
      throw new Error("simulator terminated", code);
    }
  });
  await waitOn({ resources: [`${simulatorUrl()}/api/state`], log: true });
}

export async function stopSimulator() {
  const port = workerPort();
  console.log("shutting down simulator", { port });
  await axios.post(`${simulatorUrl()}/api/shutdown`);
  console.log(`wait until port ${port} is closed`);
  await waitOn({ resources: [`tcp:localhost:${port}`], reverse: true });
}
