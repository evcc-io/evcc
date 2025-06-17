import "bootstrap/dist/css/bootstrap.min.css";
import "../css/app.css";
import { createApp, defineComponent, h } from "vue";
import { VueHeadMixin, createHead } from "@unhead/vue/client";
import App from "./views/App.vue";
import setupRouter from "./router.ts";
import setupI18n from "./i18n.ts";
import featureflags from "./featureflags.ts";
import { watchThemeChanges } from "./theme.ts";
import { appDetection, sendToApp } from "./utils/native";
import type { Notification } from "./types/evcc";

// lazy load smoothscroll polyfill. mainly for safari < 15.4
if (!window.CSS.supports("scroll-behavior", "smooth")) {
  console.log("no native smoothscroll support. polyfilling...");
  import("smoothscroll-polyfill").then((module) => {
    module.polyfill();
  });
}

const app = createApp(
  defineComponent({
    data() {
      return { notifications: [] as Notification[], offline: false };
    },
    watch: {
      offline(value) {
        console.log(`we are ${value ? "offline" : "online"}`);
      },
    },
    methods: {
      raise(msg: Notification) {
        if (this.offline) return;
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
      clear() {
        this.notifications = [];
      },
      setOnline() {
        this.offline = false;
        sendToApp({ type: "online" });
      },
      setOffline() {
        this.offline = true;
        sendToApp({ type: "offline" });
      },
    },
    render() {
      return h(App, { notifications: this.notifications, offline: this.offline });
    },
  })
);

const i18n = setupI18n();
const head = createHead();

app.use(i18n);
app.use(setupRouter(i18n.global));
app.use(featureflags);
app.use(head);
app.mixin(VueHeadMixin);
window.app = app.mount("#app");

watchThemeChanges();
appDetection();

if (window.evcc.customCss === "true") {
  const link = document.createElement("link");
  link.href = `./custom.css`;
  link.rel = "stylesheet";
  document.head.appendChild(link);
}
