import VueRouter from "vue-router";
import Vue from "vue";
import Main from "./views/Main.vue";
import Config from "./views/Config.vue";

Vue.use(VueRouter);

const routes = [
  { path: "/", component: Main },
  { path: "/config", component: Config },
];

export default new VueRouter({
  routes,
  linkExactActiveClass: "active", // Bootstrap <nav>
});
