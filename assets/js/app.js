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
      this.raise(msg);
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

new Vue({
  el: "#app",
  router,
  data: { store },
  render: (h) => h(App),
});

window.setInterval(function () {
  axios.get("health").catch(function (res) {
    res.message = "Server unavailable";
    window.toasts.error(res);
  });
}, 5000);
