import "bootstrap/dist/css/bootstrap.min.css";
import "bootstrap";
import smoothscroll from "smoothscroll-polyfill";
import "../css/app.css";
import Vue from "vue";
import VueMeta from "vue-meta";
import api from "./api";
import App from "./views/App";
import router from "./router";
import i18n from "./i18n";
import "./tooltip";
import store from "./store";

smoothscroll.polyfill();

Vue.use(VueMeta);

window.app = new Vue({
  el: "#app",
  router,
  i18n,
  data: { store, notifications: [] },
  methods: {
    raise: function (msg) {
      console[msg.type](msg);
      const now = new Date();
      const latestMsg = this.notifications[0];
      if (latestMsg && latestMsg.message === msg.message) {
        latestMsg.count++;
        latestMsg.time = now;
      } else {
        this.notifications = [
          {
            ...msg,
            count: 1,
            time: now,
          },
          ...this.notifications,
        ];
      }
    },
    clear: function () {
      this.notifications = [];
    },
    error: function (msg) {
      msg.type = "error";
      this.raise(msg);
    },
    warn: function (msg) {
      msg.type = "warn";
      this.raise(msg);
    },
  },
  render: function (h) {
    return h(App, { props: { notifications: this.notifications } });
  },
});

window.setInterval(function () {
  api.get("health").catch(function () {
    window.app.error({ message: "Server unavailable" });
  });
}, 5000);
