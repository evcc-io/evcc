import "../css/app.css";
import "bootstrap/dist/css/bootstrap.min.css";
import "popper.js";
import "bootstrap";
import Vue from "vue";
import axios from "axios";
import App from "./views/App";
import Toasts from "./components/Toasts";
import router from "./router";
import store from "./store";

const loc = window.location;
axios.defaults.baseURL =
  loc.protocol + "//" + loc.hostname + (loc.port ? ":" + loc.port : "") + loc.pathname + "api";
axios.defaults.headers.post["Content-Type"] = "application/json";

Function.prototype.throttle = function(minimumDistance) {
  let timeout,
      lastCalled = 0,
      throttledFunction = this;

  function throttleCore() {
     let context = this;

     function callThrottledFunction(args) {
        lastCalled = Date.now();
        throttledFunction.apply(context, args);
     }
     // Wartezeit bis zum nächsten Aufruf bestimmen
     let timeToNextCall = minimumDistance - (Date.now() - lastCalled);
     // Egal was kommt, einen noch offenen alten Call löschen
     cancelTimer();
     // Aufruf direkt durchführen oder um offene Wartezeit verzögern
     if (timeToNextCall < 0) {
        callThrottledFunction(arguments, 0);
     } else {
        timeout = setTimeout(callThrottledFunction, timeToNextCall, arguments);
     }
  }
  function cancelTimer() {
     if (timeout) {
        clearTimeout(timeout);
        timeout = undefined;
     }
  }
  // Aufsperre aufheben und gepeicherte Rest-Aufrufe löschen
  throttleCore.reset = function() {
     cancelTimer();
     lastCalled = 0;
  }
  return throttleCore;
};

window.toasts = new Vue({
  el: "#toasts",
  render: function (h) {
    return h(Toasts, { props: { items: this.items, count: this.count } });
  },
  data: {
    items: {},
    count: 0,
  },
  name: "ToastsRoot",
  methods: {
    raise: function (msg) {
      let found = false;
      Object.keys(this.items).forEach(function (k) {
        let m = this.items[k];
        if (m.type == msg.type && m.message == msg.message) {
          found = true;
        }
      }, this);
      if (!found) {
        msg.id = this.count++;
        Vue.set(this.items, msg.id, msg);
      }
    },
    error: function (msg) {
      msg.type = "error";
      if (msg.message == "Request failed with status code 403") {
        msg.message = "Keine Berechtigung! Bitte einloggen.";
      }
      this.raise(msg)
    },
    warn: function (msg) {
      msg.type = "warn";
      this.raise(msg);
    },
    remove: function (msg) {
      Vue.delete(this.items, msg.id);
    },
  },
});

window.throttledToasts = function() {};

new Vue({
  el: "#app",
  router,
  data: { store },
  render: (h) => h(App),
});

window.setInterval(async function () {
  if (window.throttledToasts['health'] == undefined) window.throttledToasts['health'] = window.toasts.error.throttle(30000);

  await axios.get("health")
  .then(function (res) {
    window.throttledToasts['health'].reset();
  })
  .catch(function (res) {
    res.message = "EVCC nicht erreichbar";
    window.throttledToasts['health'](res);
  });
}, 5000);
