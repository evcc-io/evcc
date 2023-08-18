import { createRouter, createWebHashHistory } from "vue-router";

import Main from "./views/Main.vue";
import ChargingSessions from "./views/ChargingSessions.vue";
import Config from "./views/Config.vue";
import { ensureCurrentLocaleMessages } from "./i18n";

export default function setupRouter(i18n) {
  const router = createRouter({
    history: createWebHashHistory(),
    routes: [
      { path: "/", component: Main, props: true },
      { path: "/config", component: Config, props: true },
      {
        path: "/sessions",
        component: ChargingSessions,
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
  router.beforeEach(async () => {
    await ensureCurrentLocaleMessages(i18n.global);
    return true;
  });
  return router;
}
