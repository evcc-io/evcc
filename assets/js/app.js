import "popper.js";
import "bootstrap";
import Vue from "vue";
import axios from "axios";
import { library } from "@fortawesome/fontawesome-svg-core";
import {
  faSun,
  faArrowUp,
  faArrowDown,
  faTemperatureLow,
  faTemperatureHigh,
  faThermometerHalf,
  faLeaf,
  faChevronUp,
  faChevronDown,
  faExclamationTriangle,
} from "@fortawesome/free-solid-svg-icons";
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
  render: (h) => h(Toasts),
}).$mount("#toasts");

new Vue({
  router,
  data: { store },
  render: (h) => h(App),
}).$mount("#app");

window.setInterval(function () {
  axios.get("health").catch(function (res) {
    res.message = "Server unavailable";
    window.toasts.error(res);
  });
}, 5000);
