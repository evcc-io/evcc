import axios from "axios";
import { openLoginModal } from "./components/Auth/auth";

const { protocol, hostname, port, pathname } = window.location;

const base = protocol + "//" + hostname + (port ? ":" + port : "") + pathname;

// override the way axios serializes arrays in query parameters (a=1&a=2&a=3 instead of a[]=1&a[]=2&a[]=3)
function customParamsSerializer(params) {
  const queryString = Object.keys(params)
    .filter((key) => params[key] !== null)
    .map((key) => {
      const value = params[key];
      if (Array.isArray(value)) {
        return value.map((v) => `${encodeURIComponent(key)}=${encodeURIComponent(v)}`).join("&");
      }
      return `${encodeURIComponent(key)}=${encodeURIComponent(value)}`;
    })
    .join("&");
  return queryString;
}

const api = axios.create({
  baseURL: base + "api/",
  headers: {
    Accept: "application/json",
  },
  paramsSerializer: customParamsSerializer,
});

// global error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // handle unauthorized errors
    if (error.response?.status === 401) {
      openLoginModal();
      return Promise.reject(error);
    }

    const message = [`${error.message}.`];
    if (error.response?.data?.error) {
      message.push(`${error.response.data.error}.`);
    }
    if (error.config) {
      const method = error.config.method.toUpperCase();
      const url = error.request.responseURL;
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

export const allowClientError = {
  validateStatus(status) {
    return status >= 200 && status < 500;
  },
};
