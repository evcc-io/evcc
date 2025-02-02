import { createRouter, createWebHashHistory } from "vue-router";
import Modal from "bootstrap/js/dist/modal";
import { ensureCurrentLocaleMessages } from "./i18n";
import { openLoginModal, statusUnknown, updateAuthStatus, isLoggedIn, isConfigured } from "./auth";

function hideAllModals() {
  [...document.querySelectorAll(".modal.show")].forEach((modal) => {
    // skip unclosable modals
    if (modal.getAttribute("data-bs-backdrop") === "static") return;

    const modalInstance = Modal.getInstance(modal);
    if (modalInstance) {
      modalInstance.hide();
    }
  });
}

async function ensureAuth(to) {
  await updateAuthStatus();
  if (!isConfigured()) {
    return false;
  }
  if (!isLoggedIn() && !statusUnknown()) {
    openLoginModal(to.path);
    return false;
  }
  return true;
}

export default function setupRouter(i18n) {
  const router = createRouter({
    history: createWebHashHistory(),
    routes: [
      { path: "/", component: () => import("./views/Main.vue"), props: true },
      {
        path: "/config",
        component: () => import("./views/Config.vue"),
        beforeEnter: ensureAuth,
        props: true,
      },
      {
        path: "/sessions",
        component: () => import("./views/Sessions.vue"),
        props: (route) => {
          const { month, year, loadpoint, vehicle, period } = route.query;
          return {
            month: month ? parseInt(month, 10) : undefined,
            year: year ? parseInt(year, 10) : undefined,
            period,
            loadpointFilter: loadpoint,
            vehicleFilter: vehicle,
          };
        },
      },
      {
        path: "/log",
        component: () => import("./views/Log.vue"),
        beforeEnter: ensureAuth,
        props: (route) => {
          const { areas, level } = route.query;
          return {
            areas: areas ? areas.split(",") : undefined,
            level,
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
