import { computed, reactive } from "vue";
import type { State } from "./types/evcc";
import { convertToUiLoadpoints } from "./uiLoadpoints";
import { useDebouncedComputed } from "./utils/useDebouncedComputed";
import settings from "./settings";

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
  loadpoints: [],
  vehicles: {},
  forecast: {},
};

const state = reactive(initialState);

// create derived loadpoints array with ui specific fields (defaults, browser settings, ...); debounce for better performance
const uiLoadpoints = useDebouncedComputed(
  () => convertToUiLoadpoints(state.loadpoints, state.vehicles),
  () => [state.loadpoints, state.vehicles, settings.loadpoints],
  50
);

// name → title lookup for loadpoints and meters
const deviceTitles = computed(() => {
  const map: Record<string, string> = {};
  const add = (e: { name?: string; title?: string } | undefined) => {
    if (e?.name && e.title) map[e.name] = e.title;
  };
  state.loadpoints?.forEach(add);
  state.pv?.forEach(add);
  state.battery?.devices?.forEach(add);
  state.aux?.forEach(add);
  state.ext?.forEach(add);
  add(state.grid);
  return map;
});

export interface Store {
  state: State; // raw state from websocket
  uiLoadpoints: typeof uiLoadpoints;
  deviceTitles: typeof deviceTitles;
  offline(value: boolean): void;
  update(msg: any): void;
  reset(): void;
}

const store: Store = {
  state,
  uiLoadpoints,
  deviceTitles,
  offline(value: boolean) {
    state.offline = value;
  },
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
