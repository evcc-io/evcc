import { createRouter, createWebHashHistory } from "vue-router";

import Main from "./views/Main.vue";
import { ensureCurrentLocaleMessages } from "./i18n";

export default function setupRouter(i18n) {
  const router = createRouter({
    history: createWebHashHistory(),
    routes: [
      { path: "/", component: Main, props: true },
      { path: "/config", component: () => import("./views/Config.vue"), props: true },
      {
        path: "/sessions",
        component: () => import("./views/ChargingSessions.vue"),
        props: (route) => {
          const { month, year, loadpoint, vehicle } = route.query;
          return {
            month: month ? parseInt(month, 10) : undefined,
            year: year ? parseInt(year, 10) : undefined,
            loadpointFilter: loadpoint,
            vehicleFilter: vehicle,
          };
        },
      },
    ],
  });
  router.beforeEach(async (to) => {
    await ensureCurrentLocaleMessages(i18n.global);
    // add config class to html element when on config page -> other background color
    window.document.documentElement.classList.toggle("config", to.path === "/config");
    return true;
  });
  return router;
}
