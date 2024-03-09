import "bootstrap/dist/css/bootstrap.min.css";
import "../css/app.css";
import { createApp, h } from "vue";
import { VueHeadMixin, createHead } from "@unhead/vue";
import App from "./views/App.vue";
import setupRouter from "./router";
import setupI18n from "./i18n";
import featureflags from "./featureflags";
import { watchThemeChanges } from "./theme";
import { appDetection, sendToApp } from "./utils/native";

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
      if (!msg.level) msg.level = "error";
      const now = new Date();
      const latestMsg = this.notifications[0];
      if (latestMsg && latestMsg.message === msg.message && latestMsg.lp === msg.lp) {
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
    setOnline: function () {
      this.offline = false;
      sendToApp({ type: "online" });
    },
    setOffline: function () {
      this.offline = true;
      sendToApp({ type: "offline" });
    },
  },
  render: function () {
    return h(App, { notifications: this.notifications, offline: this.offline });
  },
});

const i18n = setupI18n();
const head = createHead();

app.use(i18n);
app.use(setupRouter(i18n));
app.use(featureflags);
app.use(head);
app.mixin(VueHeadMixin);
window.app = app.mount("#app");

watchThemeChanges();
appDetection();
