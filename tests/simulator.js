import waitOn from "wait-on";
import axios from "axios";
import { exec } from "child_process";

export const SIMULATOR_HOST = "localhost:7072";
export const SIMULATOR_URL = `http://${SIMULATOR_HOST}/`;
const HEALTH_URL = SIMULATOR_URL + "api/state";
const SHUTDOWN_URL = SIMULATOR_URL + "api/shutdown";

export async function startSimulator() {
  console.log("starting simulator");
  const instance = exec("npm run simulator");
  console.log("exec end");
  instance.stdout.pipe(process.stdout);
  instance.stderr.pipe(process.stderr);

  instance.on("exit", (code) => {
    if (code !== 0) {
      throw new Error("simulator terminated", code);
    }
  });
  console.log("waiton");
  await waitOn({ resources: [HEALTH_URL], log: true });
}

export async function stopSimulator() {
  console.log("shutting down simulator");
  await axios.post(SHUTDOWN_URL);
}
