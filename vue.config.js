const CopyWebpackPlugin = require("copy-webpack-plugin");

module.exports = {
  pages: {
    index: { entry: "./assets/js/app.js", template: "./assets/index.html", title: "evcc" },
  },
  outputDir: "./dist",
  publicPath: "./",
  configureWebpack: {
    plugins: [new CopyWebpackPlugin([{ from: "assets/ico/", to: "ico" }])],
  },
};
