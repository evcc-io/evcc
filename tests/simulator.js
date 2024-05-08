import os from "os";
import path from "path";
import fs from "fs";
import waitOn from "wait-on";
import axios from "axios";
import { exec } from "child_process";

function port() {
  const index = process.env.TEST_PARALLEL_INDEX * 1;
  return 12000 + index;
}

export function simulatorHost() {
  return `localhost:${port()}`;
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
  console.log("starting simulator");
  const instance = exec(`npm run simulator -- --port ${port()}`);
  console.log("exec end");
  instance.stdout.pipe(process.stdout);
  instance.stderr.pipe(process.stderr);

  instance.on("exit", (code) => {
    if (code !== 0) {
      throw new Error("simulator terminated", code);
    }
  });
  console.log("waiton");
  await waitOn({ resources: [`${simulatorUrl()}/api/state`], log: true });
}

export async function stopSimulator() {
  console.log("shutting down simulator");
  await axios.post(`${simulatorUrl()}/api/shutdown`);
}
