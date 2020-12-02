import VueRouter from "vue-router";
import Vue from "vue";
import Main from "./views/Main";
import Config from "./views/Config";

Vue.use(VueRouter);

const routes = [
  { path: "/", component: Main },
  { path: "/config", component: Config },
];

module.exports = new VueRouter({
  routes,
  linkExactActiveClass: "active", // Bootstrap <nav>
});
