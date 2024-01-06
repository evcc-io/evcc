import settings from "./settings";

export function setHiddenFeatures(value) {
  settings.hiddenFeatures = value;
}

export function getHiddenFeatures() {
  return settings.hiddenFeatures;
}

export default {
  install: (app) => {
    app.config.globalProperties.$hiddenFeatures = getHiddenFeatures;
  },
};
