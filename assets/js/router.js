import { createRouter, createWebHashHistory } from "vue-router";

import Main from "./views/Main.vue";
import ChargingSessions from "./views/ChargingSessions.vue";
import { ensureCurrentLocaleMessages } from "./i18n";

export default function setupRouter(i18n) {
  const router = createRouter({
    history: createWebHashHistory(),
    routes: [
      { path: "/", component: Main, props: true },
      { path: "/sessions", component: ChargingSessions, props: true },
    ],
  });
  router.beforeEach(async () => {
    await ensureCurrentLocaleMessages(i18n.global);
    return true;
  });
  return router;
}
