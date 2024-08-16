import { reactive, watch } from "vue";
import api from "./api";
import store from "./store";
import Modal from "bootstrap/js/dist/modal";
import { isSystemError } from "./utils/fatal";

const auth = reactive({
  configured: true,
  loggedIn: null, // true / false / null (unknown)
  nextUrl: null,
});

export async function updateAuthStatus() {
  if (store.state.offline || isSystemError(store.state.fatal)) {
    // system not ready, skip auth check
    return;
  }

  try {
    const res = await api.get("/auth/status", {
      validateStatus: (code) => [200, 501, 500].includes(code),
    });
    if (res.status === 501) {
      auth.configured = false;
    }
    if (res.status === 200) {
      auth.configured = true;
      auth.loggedIn = res.data === true;
    }
    if (res.status === 500) {
      auth.loggedIn = null;
      console.log("unable to fetch auth status", res);
    }
  } catch (e) {
    console.log("unable to fetch auth status", e);
  }
}

export async function logout() {
  try {
    await api.post("/auth/logout");
    await updateAuthStatus();
    auth.nextUrl = null;
  } catch (e) {
    console.log("unable to logout", e);
  }
}

export function isLoggedIn() {
  return auth.loggedIn === true;
}

export function statusUnknown() {
  return auth.loggedIn === null;
}

export function isConfigured() {
  return auth.configured;
}

export function getAndClearNextUrl() {
  const nextUrl = auth.nextUrl;
  auth.nextUrl = null;
  return nextUrl;
}

export function openLoginModal(nextUrl = null) {
  auth.nextUrl = nextUrl;
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

let timeoutId = null;
function debounedUpdateAuthStatus() {
  clearTimeout(timeoutId);
  timeoutId = setTimeout(() => {
    updateAuthStatus();
  }, 500);
}

watch(() => store.state.offline, debounedUpdateAuthStatus);
watch(() => store.state.fatal, debounedUpdateAuthStatus);

export default auth;
