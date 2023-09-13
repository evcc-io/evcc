import "bootstrap/dist/css/bootstrap.min.css";
import "../css/app.css";
import { createApp, h } from "vue";
import { createMetaManager, plugin as metaPlugin } from "vue-meta";
import App from "./views/App.vue";
import setupRouter from "./router";
import setupI18n from "./i18n";
import featureflags from "./featureflags";
import { watchThemeChanges } from "./theme";

// lazy load smoothscroll polyfill. mainly for safari < 15.4
if (!window.CSS.supports("scroll-behavior", "smooth")) {
  console.log("no native smoothscroll support. polyfilling...");
  import("smoothscroll-polyfill").then((module) => {
    module.polyfill();
  });
}

const app = createApp({
  data() {
    return { notifications: [], offline: false };
  },
  watch: {
    offline: function (value) {
      console.log(`we are ${value ? "offline" : "online"}`);
    },
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
    setOnline: function () {
      this.offline = false;
    },
    setOffline: function () {
      this.offline = true;
    },
    warn: function (msg) {
      msg.type = "warn";
      this.raise(msg);
    },
  },
  render: function () {
    return h(App, { notifications: this.notifications, offline: this.offline });
  },
});

const i18n = setupI18n();
app.use(i18n);
app.use(setupRouter(i18n));
app.use(createMetaManager());
app.use(metaPlugin);
app.use(featureflags);
window.app = app.mount("#app");

watchThemeChanges();
