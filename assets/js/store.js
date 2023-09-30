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

const state = reactive({
  loadpoints: [], // ensure array type
});

const store = {
  state,
  update: function (msg) {
    Object.keys(msg).forEach(function (k) {
      if (k === "log") {
        window.app.raise(msg[k]);
      } else {
        setProperty(state, k.split("."), msg[k]);
      }
    });
  },
};

export default store;
