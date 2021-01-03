import VueRouter from "vue-router";
import Vue from "vue";
import Main from "./views/Main";
import Config from "./views/Config";

Vue.use(VueRouter);

const routes = [
  { path: "/", component: Main },
  { path: "/config", component: Config },
];

export default new VueRouter({
  routes,
  linkExactActiveClass: "active", // Bootstrap <nav>
});
