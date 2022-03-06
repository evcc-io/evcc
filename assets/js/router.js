import { createRouter, createWebHistory } from "vue-router";

import Main from "./views/Main.vue";

export default createRouter({
  history: createWebHistory(),
  routes: [{ path: "/", component: Main, props: true }],
});
