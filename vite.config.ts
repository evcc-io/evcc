import { defineConfig, lazyPlugins } from "vite-plus";
import vuePlugin from "@vitejs/plugin-vue";
import legacy from "@vitejs/plugin-legacy";
import { browserslistToTargets } from "lightningcss";
import browserslist from "browserslist";
import { visualizer } from "rollup-plugin-visualizer";
import path from "path";

const frontendPort = Number(process.env.VITE_PORT) || 7071;
const backendUrl = `http://localhost:${Number(process.env.VITE_BACKEND_PORT) || 7070}`;

export default defineConfig({
  run: {
    cache: {
      scripts: true,
    },
    tasks: {
      build: {
        command: "vp build",
        // node_modules layout differs between jobs that ran vitest and those that did not
        input: [{ auto: true }, "!**/node_modules/**"],
      },
      openapi: {
        command: "tsx scripts/state-schema/index.ts",
        // generated schema is an output, not an input
        input: [{ auto: true }, "!server/openapi.state.yaml"],
      },
      test: {
        command:
          "cross-env TZ=Europe/Berlin NODE_OPTIONS=--no-experimental-webstorage vp test",
        // vitest keeps its own result cache below node_modules
        input: [{ auto: true }, "!**/node_modules/.vite/vitest/**"],
      },
    },
  },
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
          "no-unused-vars": ["error", { ignoreRestSiblings: true }],
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
  server: {
    port: frontendPort,
    proxy: {
      "/api": backendUrl,
      "/i18n": backendUrl,
      "/providerauth": backendUrl,
      "/globals.js": backendUrl,
      "/custom.css": backendUrl,
      "/custom-logo-light": backendUrl,
      "/custom-logo-dark": backendUrl,
      "/ws": { target: backendUrl.replace("http", "ws"), ws: true },
    },
  },
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
