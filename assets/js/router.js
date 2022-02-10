import { createRouter, createWebHistory } from "vue-router";

import Main from "./views/Main.vue";
import Config from "./views/Config.vue";

const routes = [
  { path: "/", component: Main },
  { path: "/config", component: Config },
];

export default createRouter({
  history: createWebHistory(),
  routes,
  linkExactActiveClass: "active", // Bootstrap <nav>
});
