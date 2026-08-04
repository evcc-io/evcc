import { reactive } from "vue";
import type { Forecast, State } from "./types/evcc";
import { convertToUiLoadpoints } from "./uiLoadpoints";
import { useDebouncedComputed } from "./utils/useDebouncedComputed";
import { expandForecast } from "./utils/forecast";
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

// State types the forecast in wire format for schema generation, update()
// expands it via expandForecast, so the store carries the expanded Forecast
export type UiState = Omit<State, "forecast"> & { forecast: Forecast };

const initialState: UiState = {
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

export interface Store {
  state: UiState; // raw state from websocket, forecast expanded
  uiLoadpoints: typeof uiLoadpoints;
  offline(value: boolean): void;
  update(msg: any): void;
  reset(): void;
}

const store: Store = {
  state,
  uiLoadpoints,
  offline(value: boolean) {
    state.offline = value;
  },
  update(msg) {
    Object.keys(msg).forEach(function (k) {
      if (k === "log") {
        window.app.raise(msg[k]);
      } else {
        setProperty(state, k.split("."), expandForecast(k, msg[k]));
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
