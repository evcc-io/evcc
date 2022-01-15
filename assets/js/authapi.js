import axios from "axios";

const { protocol, hostname, port, pathname } = window.location;

const authAPI = axios.create({
  baseURL: protocol + "//" + hostname + (port ? ":" + port : "") + pathname + "auth/",
  headers: {
    Accept: "application/json",
  },
});

// global error handling
authAPI.interceptors.response.use(
  (response) => response,
  (error) => {
    const url = error.config.baseURL + error.config.url;
    const message = `${error.message}: auth API request failed ${url}`;
    window.app.error({ message });
  }
);

export default authAPI;