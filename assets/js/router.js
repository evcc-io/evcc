import { createRouter, createWebHashHistory } from "vue-router";

import Main from "./views/Main.vue";

export default createRouter({
  history: createWebHashHistory(),
  routes: [{ path: "/", component: Main, props: true }],
});
