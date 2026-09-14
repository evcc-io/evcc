import { defineConfig } from "vite-plus";
import vue from "@vitejs/plugin-vue";
import api from "./api";

export default defineConfig({
  plugins: [vue(), api()],
  server: { port: 7072, host: true },
});
