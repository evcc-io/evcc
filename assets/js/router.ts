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
import { initConfigModal } from "./configModal";
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

// Custom stringifyQuery to keep brackets unencoded in URLs
export function stringifyQuery(query?: Record<string, any>): string {
  if (!query) return "";
  const parts: string[] = [];
  for (const key of Object.keys(query)) {
    const value = query[key];
    if (Array.isArray(value)) {
      for (const v of value) {
        parts.push(v ? `${key}=${encodeURIComponent(v)}` : key);
      }
    } else if (value) {
      parts.push(`${key}=${encodeURIComponent(value)}`);
    } else {
      parts.push(key);
    }
  }
  return parts.join("&");
}

export default function setupRouter(i18n: VueI18nInstance) {
  const router = createRouter({
    history: createWebHashHistory(),
    stringifyQuery,
    routes: [
      {
        path: "/",
        component: () => import("./views/Main.vue"),
        props: (route) => {
          const { lp } = route.query;
          return {
            selectedLoadpointId: lp as string | undefined,
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
        path: "/optimize",
        component: () => import("./views/Optimize.vue"),
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
      {
        path: "/issue",
        component: () => import("./views/Issue.vue"),
        beforeEnter: ensureAuth,
        props: true,
      },
    ],
  });
  router.beforeEach(async () => {
    await ensureCurrentLocaleMessages(i18n);
    return true;
  });
  router.afterEach((to, from) => {
    // Only hide modals when the actual route path changes, not query parameters
    if (to.path !== from.path) {
      hideAllModals();
    }
  });
  initConfigModal(router);
  return router;
}
