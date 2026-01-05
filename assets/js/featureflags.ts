import type { App } from "vue";
import settings from "./settings";

export function setHiddenFeatures(value: boolean) {
  settings.hiddenFeatures = value;
}

export function getHiddenFeatures() {
  return settings.hiddenFeatures;
}

export default {
  install: (app: App) => {
    app.config.globalProperties.$hiddenFeatures = getHiddenFeatures;
  },
};
