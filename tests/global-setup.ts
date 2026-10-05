import { execFileSync } from "child_process";

// workers serve the prebuilt simulator, a dev server per worker is too slow on a loaded runner
export default function globalSetup() {
  execFileSync("vp", ["build", "tests/simulator"], { stdio: "inherit" });
}
