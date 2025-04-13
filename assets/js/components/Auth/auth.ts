import { reactive, watch } from "vue";
import api from "../../api.js";
import store from "../../store.js";
import Modal from "bootstrap/js/dist/modal";
import { isSystemError } from "../../utils/fatal.js";

const auth = reactive({
	configured: true,
	loggedIn: null as boolean | null, // true / false / null (unknown)
	nextUrl: null as string | null, // url to navigate to after login
	nextModal: null as Modal | null, // modal instance to show after login
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

export function getAndClearNextModal() {
	const nextModal = auth.nextModal;
	auth.nextModal = null;
	return nextModal;
}

export function openLoginModal(nextUrl = null, nextModal = null) {
	auth.nextUrl = nextUrl;
	auth.nextModal = nextModal;
	const modal = Modal.getOrCreateInstance(document.getElementById("loginModal") as HTMLElement);
	modal.show();
}

// show/hide password modal based on auth status
watch(
	() => auth.configured,
	(configured) => {
		console.log("configured", configured);
		const modal = Modal.getOrCreateInstance(
			document.getElementById("passwordModal") as HTMLElement
		);
		if (configured) {
			modal.hide();
		} else {
			modal.show();
		}
	}
);

let timeoutId: number | undefined = undefined;
function debounedUpdateAuthStatus() {
	clearTimeout(timeoutId);
	timeoutId = setTimeout(() => {
		updateAuthStatus();
	}, 500);
}

// update auth status on reconnect or server restart
watch(() => store.state.offline, debounedUpdateAuthStatus);
watch(() => store.state.startup, debounedUpdateAuthStatus);

export default auth;
