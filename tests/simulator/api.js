import bodyParser from "body-parser";

let state = {
  site: {
    grid: { power: 0 },
    pv: { power: 0 },
    battery: { power: 0, soc: 0 },
  },
  loadpoints: [{ power: 0, energy: 0, enabled: false, status: "A" }],
  vehicles: [{ soc: 0, range: 0 }],
};

const stateApiMiddleware = (req, res, next) => {
  if (req.method === "POST" && req.originalUrl === "/api/state") {
    console.log("POST /api/state", req.body);
    state = req.body;
    res.end();
  } else if (req.method === "POST" && req.originalUrl === "/api/shutdown") {
    console.log("POST /api/shutdown", req.body);
    res.end();
    process.exit();
  } else if (req.originalUrl === "/api/state") {
    res.end(JSON.stringify(state));
  } else {
    next();
  }
};

const openemsMiddleware = (req, res, next) => {
  const endpoints = {
    "/rest/channel/_sum/GridActivePower": { value: state.site.grid.power },
    "/rest/channel/_sum/ProductionActivePower": { value: state.site.pv.power },
    "/rest/channel/_sum/EssDischargePower": { value: state.site.battery.power },
    "/rest/channel/_sum/EssSoc": { value: state.site.battery.soc },
  };
  const endpoint = endpoints[req.originalUrl];
  if (req.method === "GET" && endpoint) {
    console.log("GET", req.originalUrl, endpoint);
    res.end(JSON.stringify(endpoint));
  } else {
    next();
  }
};

export default () => ({
  name: "api",
  enforce: "pre",
  configureServer(server) {
    console.log("configureServer");
    return () => {
      server.middlewares.use(bodyParser.json());
      server.middlewares.use(stateApiMiddleware);
      server.middlewares.use(openemsMiddleware);
    };
  },
});
