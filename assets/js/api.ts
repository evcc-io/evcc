import axios, { type AxiosResponse } from "axios";
import { openLoginModal } from "./components/Auth/auth";

const { protocol, hostname, port, pathname } = window.location;

const base = protocol + "//" + hostname + (port ? ":" + port : "") + pathname;

// override the way axios serializes arrays in query parameters (a=1&a=2&a=3 instead of a[]=1&a[]=2&a[]=3)
function customParamsSerializer(params: { [key: string]: any }) {
  return Object.keys(params)
    .filter((key) => params[key] !== null)
    .map((key) => {
      const value = params[key];
      if (Array.isArray(value)) {
        return value.map((v) => `${encodeURIComponent(key)}=${encodeURIComponent(v)}`).join("&");
      }
      return `${encodeURIComponent(key)}=${encodeURIComponent(value)}`;
    })
    .join("&");
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
    Accept: "application/json",
  },
});

export const allowClientError = {
  validateStatus(status: number) {
    return status >= 200 && status < 500;
  },
};

export function downloadFile(res: AxiosResponse) {
  if (res.status === 200) {
    const blob = new Blob([res.data]);
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;

    // Try to get filename from Content-Disposition header
    let filename = "evcc.txt";
    const disposition = res.headers["content-disposition"];
    if (disposition && disposition.indexOf("filename=") !== -1) {
      filename = disposition.split("filename=")[1].replace(/['"]/g, "").trim();
    }

    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }
}
