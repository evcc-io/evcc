import fs from "fs";
import { exec } from "child_process";

export function execEvcc(config, database) {
  const restore = database ? `sqlite3 ./evcc.db < tests/${database} && ` : "";
  console.log(restore);
  const server = exec(`${restore}EVCC_DATABASE_DSN=./evcc.db ./evcc --config tests/${config}`);
  server.stdout.pipe(process.stdout);
  return server;
}

export function stopEvcc(server) {
  server.kill();
  fs.existsSync("./evcc.db") && fs.unlinkSync("./evcc.db");
}
