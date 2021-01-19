import VueRouter from "vue-router";
import Vue from "vue";
import Main from "./views/Main";
import Config from "./views/Config";
import Setup from "./views/Setup";
import Config2 from "./testing/views/Config";

Vue.use(VueRouter);

const routes = [
  { path: "/", component: Main },
  { path: "/config", component: Config },
  { path: "/setup", component: Setup },
  { path: "/testing", component: Config2 },
];

export default new VueRouter({
  routes,
  linkExactActiveClass: "active", // Bootstrap <nav>
});
