import axios from "axios";

const api = axios.create({
  baseURL: "https://api.evcc.io/v1/",
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
