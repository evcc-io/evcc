import { reactive } from "vue";

function setProperty(obj, props, value) {
  const prop = props.shift();
  if (!obj[prop]) {
    obj[prop] = {};
  }

  if (!props.length) {
    if (value && typeof value === "object" && !Array.isArray(value)) {
      obj[prop] = { ...obj[prop], ...value };
    } else {
      obj[prop] = value;
    }
    return;
  }

  setProperty(obj[prop], props, value);
}

const initialState = {
  offline: false,
  loadpoints: [], // ensure array type
};

const state = reactive(initialState);

const store = {
  state,
  offline: function (value) {
    state.offline = value;
  },
  update: function (msg) {
    Object.keys(msg).forEach(function (k) {
      if (k === "log") {
        window.app.raise(msg[k]);
      } else {
        setProperty(state, k.split("."), msg[k]);
      }
    });
  },
  reset: function () {
    console.log("resetting state");
    // remove additional properties
    Object.keys(state).forEach(function (k) {
      if (!initialState[k]) {
        delete state[k];
      }
    });
    // reset to initial state
    Object.keys(initialState).forEach(function (k) {
      state[k] = initialState[k];
    });
  },
};

export default store;
