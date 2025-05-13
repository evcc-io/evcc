import { reactive } from "vue";
import type { State } from "./types/evcc";

function setProperty(obj: object, props: string[], value: any) {
	const prop = props.shift();
	// @ts-expect-error no-explicit-any
	if (!obj[prop]) {
		// @ts-expect-error no-explicit-any
		obj[prop] = {};
	}

	if (!props.length) {
		if (value && typeof value === "object" && !Array.isArray(value)) {
			// @ts-expect-error no-explicit-any
			obj[prop] = { ...obj[prop], ...value };
		} else {
			// @ts-expect-error no-explicit-any
			obj[prop] = value;
		}
		return;
	}

	// @ts-expect-error no-explicit-any
	setProperty(obj[prop], props, value);
}

const initialState: State = {
	offline: false,
	loadpoints: [], // ensure array type
};

const state = reactive(initialState);

const store = {
	state,
	offline(value: boolean) {
		state.offline = value;
	},
	// @ts-expect-error no-explicit-any
	update(msg) {
		Object.keys(msg).forEach(function (k) {
			if (k === "log") {
				window.app.raise(msg[k]);
			} else {
				setProperty(state, k.split("."), msg[k]);
			}
		});
	},
	reset() {
		console.log("resetting state");
		// reset to initial state
		Object.keys(initialState).forEach(function (k) {
			if (k === "offline") return;

			// @ts-expect-error no-explicit-any
			if (Array.isArray(initialState[k])) {
				// @ts-expect-error no-explicit-any
				state[k] = [];
			} else {
				// @ts-expect-error no-explicit-any
				state[k] = undefined;
			}
		});
	},
};

export default store;
