import { exec } from "child_process";

export function execEvcc(config) {
  const server = exec(`./evcc --config tests/${config}`);
  server.stdout.pipe(process.stdout);
  return server;
}
