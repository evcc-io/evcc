import "bootstrap/dist/css/bootstrap.min.css";
import "bootstrap";
import smoothscroll from "smoothscroll-polyfill";
import "../css/app.css";
import { createApp, h } from "vue";
import { createMetaManager, plugin as metaPlugin } from "vue-meta";
import api from "./api";
import App from "./views/App.vue";
import router from "./router";
import i18n from "./i18n";
import featureflags from "./featureflags";

smoothscroll.polyfill();

const app = createApp({
  data() {
    return { notifications: [] };
  },
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
  render: function () {
    return h(App, { notifications: this.notifications });
  },
});

app.use(i18n);
app.use(router);
app.use(createMetaManager());
app.use(metaPlugin);
app.use(featureflags);
window.app = app.mount("#app");

window.setInterval(function () {
  api.get("health").catch(function () {
    window.app.error({ message: "Server unavailable" });
  });
}, 5000);
