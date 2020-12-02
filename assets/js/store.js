import Vue from "vue";

function setProperty(obj, props, value) {
  const prop = props.shift();
  if (!obj[prop]) {
    Vue.set(obj, prop, {});
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

const store = {
  state: {
    loadpoints: [], // ensure array type
  },
  update: function (msg) {
    Object.keys(msg).forEach(function (k) {
      if (typeof window.toasts[k] === "function") {
        window.toasts[k]({ message: msg[k] });
      } else {
        setProperty(store.state, k.split("."), msg[k]);
      }
    });
  },
};

module.exports = store;
