import { createRouter, createWebHashHistory } from "vue-router";
import Modal from "bootstrap/js/dist/modal";
import Main from "./views/Main.vue";
import { ensureCurrentLocaleMessages } from "./i18n";
import { openLoginModal, updateAuthStatus, isLoggedIn } from "./auth";

function hideAllModals() {
  [...document.querySelectorAll(".modal.show")].forEach((modal) => {
    const modalInstance = Modal.getInstance(modal);
    if (modalInstance) {
      modalInstance.hide();
    }
  });
}

export default function setupRouter(i18n) {
  const router = createRouter({
    history: createWebHashHistory(),
    routes: [
      { path: "/", component: Main, props: true },
      {
        path: "/config",
        component: () => import("./views/Config.vue"),
        beforeEnter: async () => {
          await updateAuthStatus();
          if (!isLoggedIn()) {
            openLoginModal();
            return false;
          }
        },
        props: true,
      },
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
  router.beforeEach(async () => {
    await ensureCurrentLocaleMessages(i18n.global);
    return true;
  });
  router.afterEach(() => {
    hideAllModals();
  });
  return router;
}
