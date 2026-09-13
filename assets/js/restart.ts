import { reactive } from "vue";
import api from "./api";

const restart = reactive({
  restartNeeded: false,
  restarting: false,
});

export async function performRestart() {
  restart.restarting = true;
  try {
    await api.post("/system/shutdown");
  } catch (e: any) {
    // connection may drop before response
    if (e.response?.status < 500) {
      restart.restarting = false;
      alert(`Unable to restart server. ${e}`);
    }
  }
}

export function restartComplete() {
  restart.restarting = false;
  restart.restartNeeded = false;
}

export function showRestarting() {
  restart.restarting = true;
}

export default restart;
