import { reactive, watch } from "vue";
import api from "./api";
import Modal from "bootstrap/js/dist/modal";

const auth = reactive({
  configured: true,
  loggedIn: false,
});

export async function updateAuthStatus() {
  try {
    const res = await api.get("/auth/status", {
      validateStatus: (code) => [200, 501].includes(code),
    });
    if (res.status === 501) {
      auth.configured = false;
    }
    if (res.status === 200) {
      auth.configured = true;
      auth.loggedIn = res.data === true;
    }
  } catch (e) {
    console.log("unable to fetch auth status", e);
  }
}

export async function logout() {
  try {
    await api.post("/auth/logout");
    await updateAuthStatus();
  } catch (e) {
    console.log("unable to logout", e);
  }
}

export function isLoggedIn() {
  return auth.loggedIn;
}

export function isConfigured() {
  return auth.configured;
}

export function openLoginModal() {
  const modal = Modal.getOrCreateInstance(document.getElementById("loginModal"));
  modal.show();
}

// show/hide password modal based on auth status
watch(
  () => auth.configured,
  (configured) => {
    const modal = Modal.getOrCreateInstance(document.getElementById("passwordModal"));
    configured ? modal.hide() : modal.show();
  }
);

export default auth;
