import { createRouter, createWebHashHistory } from "vue-router";

import Main from "./views/Main.vue";
import ChargingSessions from "./views/ChargingSessions.vue";

export default createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: "/", component: Main, props: true },
    { path: "/sessions", component: ChargingSessions, props: true },
  ],
});
