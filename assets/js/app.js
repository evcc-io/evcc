import "popper.js";
import "bootstrap";
import Vue from "vue";
import axios from "axios";
import { library } from "@fortawesome/fontawesome-svg-core";
import { faSun } from "@fortawesome/free-solid-svg-icons/faSun";
import { faArrowUp } from "@fortawesome/free-solid-svg-icons/faArrowUp";
import { faArrowDown } from "@fortawesome/free-solid-svg-icons/faArrowDown";
import { faTemperatureLow } from "@fortawesome/free-solid-svg-icons/faTemperatureLow";
import { faTemperatureHigh } from "@fortawesome/free-solid-svg-icons/faTemperatureHigh";
import { faThermometerHalf } from "@fortawesome/free-solid-svg-icons/faThermometerHalf";
import { faLeaf } from "@fortawesome/free-solid-svg-icons/faLeaf";
import { faChevronUp } from "@fortawesome/free-solid-svg-icons/faChevronUp";
import { faChevronDown } from "@fortawesome/free-solid-svg-icons/faChevronDown";
import { faExclamationTriangle } from "@fortawesome/free-solid-svg-icons/faExclamationTriangle";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import App from "./views/App";
import Toasts from "./components/Toasts";
import router from "./router";
import store from "./store";

library.add(
  faSun,
  faArrowUp,
  faArrowDown,
  faTemperatureLow,
  faTemperatureHigh,
  faThermometerHalf,
  faLeaf,
  faChevronUp,
  faChevronDown,
  faExclamationTriangle
);

Vue.component("font-awesome-icon", FontAwesomeIcon);

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
