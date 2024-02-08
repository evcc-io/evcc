import axios from "axios";

const { protocol, hostname, port, pathname } = window.location;

const base = protocol + "//" + hostname + (port ? ":" + port : "") + pathname;

const api = axios.create({
  baseURL: base + "api/",
  headers: {
    Accept: "application/json",
  },
});

// global error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const message = [`${error.message}.`];
    if (error.response?.data?.error) {
      message.push(`${error.response.data.error}.`);
    }
    if (error.config) {
      const method = error.config.method.toUpperCase();
      const url = error.config.baseURL + error.config.url;
      message.push(`${method} ${url}`);
    }
    window.app.raise({ message });
    return Promise.reject(error);
  }
);

export default api;

export const i18n = axios.create({
  baseURL: base + "i18n/",
  headers: {
    Accept: "application/toml",
  },
});
