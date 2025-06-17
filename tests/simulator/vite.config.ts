import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import api from "./api";

export default defineConfig({
  plugins: [
    vue(),
    // @ts-expect-error the in api() declared 'enforce' property is not supported in TypeScript but in JavaScript
    api(),
  ],
  server: { port: 7072, host: true },
});
