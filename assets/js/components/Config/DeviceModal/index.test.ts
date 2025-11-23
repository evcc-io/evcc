import { describe, expect, it } from "vitest";
import { createServiceEndpoints, type TemplateParam } from "./index";

const buildParam = (name: string, service?: string): TemplateParam => ({
  Name: name,
  Required: false,
  Advanced: false,
  Deprecated: false,
  Service: service,
});

describe("createServiceEndpoints", () => {
  it("skips params without service", () => {
    const params = [buildParam("home", "homes"), buildParam("power", "homes/{home}/sensors")];
    const endpoints = createServiceEndpoints(params);
    expect(endpoints.map((endpoint) => endpoint.name)).toEqual(["home", "power"]);
  });

  it("replaces single placeholder", () => {
    const params = [buildParam("home", "homes"), buildParam("power", "homes/{home}/sensors")];
    const endpoints = createServiceEndpoints(params);
    const homeEndpoint = endpoints.find(({ name }) => name === "home")!;
    const powerEndpoint = endpoints.find(({ name }) => name === "power")!;
    expect(homeEndpoint.dependencies).toEqual([]);
    expect(homeEndpoint.url({})).toBe("homes");
    expect(powerEndpoint.dependencies).toEqual(["home"]);
    expect(powerEndpoint.url({ home: "main" })).toBe("homes/main/sensors");
    expect(powerEndpoint.url({ home: "with space" })).toBe("homes/with%20space/sensors");
    expect(powerEndpoint.url({} as Record<string, string>)).toBe("homes/{home}/sensors");
  });

  it("replaces multiple placeholders", () => {
    const params = [
      buildParam("home", "homes"),
      buildParam("sensor", "homes/{home}/sensors/{sensor}"),
    ];
    const endpoints = createServiceEndpoints(params);
    const sensorEndpoint = endpoints.find(({ name }) => name === "sensor")!;
    expect(sensorEndpoint.dependencies).toEqual(["home", "sensor"]);
    expect(sensorEndpoint.url({ home: "hq", sensor: "battery" })).toBe("homes/hq/sensors/battery");
  });

  it("encodes replacements", () => {
    const params = [buildParam("token", "homes/{home}/sensors/{sensor}?token={token}")];
    const endpoints = createServiceEndpoints(params);
    const tokenEndpoint = endpoints[0]!;
    expect(tokenEndpoint.url({ home: "hq", sensor: "bat/tery", token: "a+b c" })).toBe(
      "homes/hq/sensors/bat%2Ftery?token=a%2Bb%20c"
    );
    expect(tokenEndpoint.url({} as Record<string, string>)).toBe(
      "homes/{home}/sensors/{sensor}?token={token}"
    );
  });
});
