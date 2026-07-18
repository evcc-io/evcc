import "bootstrap/dist/css/bootstrap.min.css";
import "../css/app.css";
import Dropdown from "bootstrap/js/dist/dropdown";
import { createApp, defineComponent, h } from "vue";
import { VueHeadMixin, createHead } from "@unhead/vue/client";
import App from "./views/App.vue";
import setupRouter from "./router.ts";
import setupI18n from "./i18n.ts";
import { watchThemeChanges } from "./theme.ts";
import { applyUrlSettings } from "./urlSettings.ts";
import { appDetection, sendToApp } from "./utils/native";
import store from "./store";
import type { Notification } from "./types/evcc";

// keep dropdown menus out of the fixed bottom tab bar and safe-area inset
Dropdown.Default.popperConfig = (defaultConfig) => {
  const bottom = document.querySelector<HTMLElement>(".bottom-tab-bar")?.offsetHeight ?? 0;
  return {
    ...defaultConfig,
    modifiers: [
      ...(defaultConfig.modifiers ?? []),
      { name: "flip", options: { padding: { bottom } } },
      { name: "preventOverflow", options: { padding: { bottom } } },
    ],
  };
};

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
      return { notifications: [] as Notification[], wsOffline: false };
    },
    computed: {
      offline(): boolean {
        return this.wsOffline || store.state.apiReady !== true;
      },
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
        const existingMsg = this.notifications.find(
          (n: Notification) => n.message === msg.message && n.lp === msg.lp
        );
        if (existingMsg) {
          existingMsg.count++;
          existingMsg.time = now;
          // move to front
          this.notifications = [
            existingMsg,
            ...this.notifications.filter((n: Notification) => n !== existingMsg),
          ];
        } else {
          this.notifications = [
            { ...msg, count: 1, time: now },
            ...this.notifications.slice(0, 14), // keep only last 15
          ];
        }
      },
      clear() {
        this.notifications = [];
      },
      setOnline() {
        this.wsOffline = false;
        sendToApp({ type: "online" });
      },
      setOffline() {
        this.wsOffline = true;
        sendToApp({ type: "offline" });
      },
    },
    render() {
      return h(App, { notifications: this.notifications, offline: this.offline });
    },
  })
);

applyUrlSettings();

const i18n = setupI18n();
const head = createHead();
const router = setupRouter(i18n.global);

app.use(i18n);
app.use(router);
app.use(head);
app.mixin(VueHeadMixin);
window.app = app.mount("#app");

watchThemeChanges();
appDetection(router);

if (window.evcc.customCss === "true") {
  const link = document.createElement("link");
  link.href = `./custom.css`;
  link.rel = "stylesheet";
  document.head.appendChild(link);
}
