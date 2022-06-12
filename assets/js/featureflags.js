export default {
  install: (app) => {
    app.config.globalProperties.$hiddenFeatures = window.localStorage["hidden_features"] === "true";
  },
};
