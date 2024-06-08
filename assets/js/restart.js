import { reactive } from "vue";
import api from "./api";

const restart = reactive({
  restartNeeded: false,
  restarting: false,
});

export async function performRestart() {
  try {
    await api.post("/system/shutdown");
    restart.restarting = true;
  } catch (e) {
    alert("Unabled to restart server.");
  }
}

export function restartComplete() {
  restart.restarting = false;
  restart.restartNeeded = false;
}

export default restart;
