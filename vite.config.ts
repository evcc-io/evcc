import { defineConfig, lazyPlugins } from "vite-plus";
import vuePlugin from "@vitejs/plugin-vue";
import legacy from "@vitejs/plugin-legacy";
import { browserslistToTargets } from "lightningcss";
import browserslist from "browserslist";
import { visualizer } from "rollup-plugin-visualizer";
import path from "path";

export default defineConfig({
  staged: {
    "*": "vp check --fix",
  },
  lint: {
    plugins: ["oxc", "typescript", "unicorn", "react"],
    categories: {
      correctness: "warn",
    },
    env: {
      builtin: true,
    },
    overrides: [
      {
        files: ["assets/**/*.{ts,js,vue}", "tests/**/*.ts"],
        rules: {
          "no-param-reassign": "error",
          "vue/require-default-prop": "off",
          "vue/no-reserved-component-names": "off",
          "typescript/no-explicit-any": "off",
        },
        plugins: ["vue"],
        env: {
          es2026: true,
          browser: true,
          node: true,
        },
      },
    ],
    options: {},
    jsPlugins: [
      {
        name: "vite-plus",
        specifier: "vite-plus/oxlint-plugin",
      },
    ],
    rules: {
      "vite-plus/prefer-vite-plus-imports": "error",
    },
  },
  fmt: {
    printWidth: 100,
    trailingComma: "es5",
    sortPackageJson: false,
    ignorePatterns: ["tests/custom-css.css"],
    overrides: [{ files: ["**/*.vue"], options: { tabWidth: 4 } }],
  },
  root: "./assets",
  publicDir: "public",
  base: "./",
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./assets/js"),
    },
  },
  css: {
    transformer: "lightningcss",
    lightningcss: {
      drafts: { customMedia: true },
      targets: browserslistToTargets(browserslist()),
    },
  },
  build: {
    outDir: "../dist/",
    emptyOutDir: true,
    assetsInlineLimit: 1024,
    chunkSizeWarningLimit: 800, // legacy build increases file size
  },
  server: (() => {
    const frontend = Number(process.env.VITE_PORT) || 7071;
    const backend = Number(process.env.VITE_BACKEND_PORT) || 7070;
    return {
      port: frontend,
      proxy: {
        "/api": `http://localhost:${backend}`,
        "/i18n": `http://localhost:${backend}`,
        "/providerauth": `http://localhost:${backend}`,
        "/ws": { target: `ws://localhost:${backend}`, ws: true },
      },
    };
  })(),
  plugins: lazyPlugins(() => [
    legacy({
      modernPolyfills: ["es.promise.all-settled"],
    }),
    vuePlugin({
      template: {
        compilerOptions: {
          isCustomElement: (tag) => tag.startsWith("shopicon-"),
        },
      },
    }),
    visualizer({ filename: "asset-stats.html" }),
  ]),
});
