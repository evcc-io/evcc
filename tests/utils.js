import { exec } from "child_process";

export async function execEvcc(config) {
  return await exec(`./evcc --config tests/${config}`, (error, stdout, stderr) => {
    if (error) {
      console.error(`error: ${error}`);
      return;
    }
    console.log(`stdout: ${stdout}`);
    console.error(`stderr: ${stderr}`);
  });
}
