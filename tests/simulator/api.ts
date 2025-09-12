import bodyParser from "body-parser";
import type { Connect, ViteDevServer } from "vite";
import type { ServerResponse } from "http";

let state = {
  site: {
    grid: { power: 0, energy: 0 },
    pv: { power: 0, energy: 0 },
    battery: { power: 0, soc: 0 },
  },
  loadpoints: [{ power: 0, energy: 0, enabled: false, status: "A" }],
  vehicles: [{ soc: 0, range: 0 }],
  hems: { relay: false },
};

const loggingMiddleware = (
  req: Connect.IncomingMessage,
  _: ServerResponse,
  next: Connect.NextFunction
) => {
  console.log(`[simulator] ${req.method} ${req.url}`);
  next();
};

const stateApiMiddleware = (
  req: Connect.IncomingMessage,
  res: ServerResponse,
  next: Connect.NextFunction
) => {
  if (req.method === "POST" && req.originalUrl === "/api/state") {
    console.log("[simulator] POST /api/state");
    // @ts-expect-error Property 'body' does not exist on type 'IncomingMessage'
    state = req.body;
    res.end();
  } else if (req.method === "POST" && req.originalUrl === "/api/shutdown") {
    console.log("[simulator] POST /api/shutdown");
    res.end();
    process.exit();
  } else if (req.originalUrl === "/api/state") {
    res.end(JSON.stringify(state));
  } else {
    next();
  }
};

const openemsMiddleware = (
  req: Connect.IncomingMessage,
  res: ServerResponse,
  next: Connect.NextFunction
) => {
  const endpoints = {
    "/rest/channel/_sum/GridActivePower": { value: state.site.grid.power },
    "/rest/channel/_sum/GridBuyActiveEnergy": { value: state.site.grid.energy },
    "/rest/channel/_sum/ProductionActivePower": { value: state.site.pv.power },
    "/rest/channel/_sum/ProductionActiveEnergy": { value: state.site.pv.energy },
    "/rest/channel/_sum/EssDischargePower": { value: state.site.battery.power },
    "/rest/channel/_sum/EssSoc": { value: state.site.battery.soc },
  };

  const endpoint = endpoints[req.originalUrl as keyof typeof endpoints];
  if (req.method === "GET" && endpoint) {
    console.log("[simulator] GET", req.originalUrl);
    res.end(JSON.stringify(endpoint));
  } else {
    next();
  }
};

const teslaloggerMiddleware = (
  req: Connect.IncomingMessage,
  res: ServerResponse,
  next: Connect.NextFunction
) => {
  if (req.method === "GET" && req.originalUrl && req.originalUrl.startsWith("/currentjson/")) {
    const idPart = req.originalUrl.split("/")[2];
    const id = idPart ? parseInt(idPart) : 0;
    const vehicle = state.vehicles[id - 1];
    if (!vehicle) {
      res.statusCode = 404;
      res.end(JSON.stringify({ error: "Vehicle not found" }));
      return;
    }
    const data = {
      battery_level: vehicle.soc,
      battery_range_km: vehicle.range,
      plugged_in: true,
      charging: false,
      odometer: 10000,
      is_preconditioning: false,
      charge_current_request: 10,
    };
    res.end(JSON.stringify(data));
  } else {
    next();
  }
};

const shellyMiddleware = (
  req: Connect.IncomingMessage,
  res: ServerResponse,
  next: Connect.NextFunction
) => {
  // simulate a shelly gen2 switch device api. implement power and energy
  if (req.originalUrl === "/shelly") {
    res.end(JSON.stringify({ gen: 2 }));
  } else if (req.originalUrl === "/rpc/Shelly.ListMethods") {
    res.end(JSON.stringify({ methods: ["Switch.GetStatus"] }));
  } else if (req.originalUrl === "/rpc/Switch.GetStatus") {
    res.end(
      JSON.stringify({
        apower: state.site.pv.power,
        aenergy: { total: state.site.pv.energy },
      })
    );
  } else {
    next();
  }
};

export default () => ({
  name: "api",
  enforce: "pre",
  configureServer(server: ViteDevServer) {
    console.log("[simulator] configured");
    return () => {
      server.middlewares.use(loggingMiddleware);
      server.middlewares.use(bodyParser.json());
      server.middlewares.use(stateApiMiddleware);
      server.middlewares.use(openemsMiddleware);
      server.middlewares.use(teslaloggerMiddleware);
      server.middlewares.use(shellyMiddleware);
    };
  },
});
