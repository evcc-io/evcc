import axios from "axios";

const { protocol, hostname, port, pathname } = window.location;

const api = axios.create({
  baseURL: protocol + "//" + hostname + (port ? ":" + port : "") + pathname + "api/",
  headers: {
    Accept: "application/json",
  },
});

// global error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const url = error.config.baseURL + error.config.url;
    const message = `${error.message}: API request failed ${url}`;
    window.app.error({ message });
    return Promise.reject(error);
  }
);
export default api;
