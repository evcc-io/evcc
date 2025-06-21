import {
  createRouter,
  createWebHashHistory,
  type RouteLocationNormalizedGeneric,
} from "vue-router";
import Modal from "bootstrap/js/dist/modal";
import { ensureCurrentLocaleMessages } from "./i18n.ts";
import {
  openLoginModal,
  statusUnknown,
  updateAuthStatus,
  isLoggedIn,
  isConfigured,
} from "./components/Auth/auth";
import type { VueI18nInstance } from "vue-i18n";

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

async function ensureAuth(to: RouteLocationNormalizedGeneric) {
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

export default function setupRouter(i18n: VueI18nInstance) {
  const router = createRouter({
    history: createWebHashHistory(),
    routes: [
      {
        path: "/",
        component: () => import("./views/Main.vue"),
        props: (route) => {
          const { lp } = route.query;
          return {
            selectedLoadpointIndex: lp ? parseInt(lp as string, 10) - 1 : undefined,
          };
        },
      },
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
            month: month ? parseInt(month as string, 10) : undefined,
            year: year ? parseInt(year as string, 10) : undefined,
            period,
            loadpointFilter: loadpoint,
            vehicleFilter: vehicle,
          };
        },
      },
      {
        path: "/energy",
        component: () => import("./views/Energy.vue"),
        props: true,
      },
      {
        path: "/log",
        component: () => import("./views/Log.vue"),
        beforeEnter: ensureAuth,
        props: (route) => {
          const { areas, level } = route.query;
          return {
            areas: typeof areas === "string" ? areas.split(",") : undefined,
            level: typeof level === "string" ? level : undefined,
          };
        },
      },
    ],
  });
  router.beforeEach(async () => {
    await ensureCurrentLocaleMessages(i18n);
    return true;
  });
  router.afterEach(() => {
    hideAllModals();
  });
  return router;
}
