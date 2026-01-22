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
    expect(homeEndpoint.url({})).toBe("homes");
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

  it("expands {modbus} for TCP/IP", () => {
    const params = [buildParam("param", "service?address=100&{modbus}")];
    const endpoints = createServiceEndpoints(params);

    expect(endpoints[0]!.url({ host: "192.168.1.1", port: "502", id: "1" })).toBe(
      "service?address=100&uri=192.168.1.1:502&id=1"
    );
  });

  it("expands {modbus} for serial", () => {
    const params = [buildParam("param", "service?address=100&{modbus}")];
    const endpoints = createServiceEndpoints(params);

    expect(
      endpoints[0]!.url({ device: "/dev/ttyUSB0", baudrate: "9600", comset: "8N1", id: "1" })
    ).toBe("service?address=100&device=%2Fdev%2FttyUSB0&baudrate=9600&comset=8N1&id=1");
  });

  it("leaves {modbus} unexpanded when connection missing", () => {
    const params = [buildParam("param", "service?address=100&{modbus}")];
    const endpoints = createServiceEndpoints(params);

    expect(endpoints[0]!.url({})).toBe("service?address=100&{modbus}");
  });

  it("prefers device over host when both present", () => {
    const params = [buildParam("param", "service?{modbus}")];
    const endpoints = createServiceEndpoints(params);

    expect(
      endpoints[0]!.url({
        device: "/dev/ttyUSB0",
        baudrate: "9600",
        comset: "8N1",
        host: "192.168.1.1",
        port: "502",
        id: "1",
      })
    ).toBe("service?device=%2Fdev%2FttyUSB0&baudrate=9600&comset=8N1&id=1");
  });

  it("treats empty strings as missing values", () => {
    const params = [buildParam("sensor", "homes/{home}/sensors")];
    const endpoints = createServiceEndpoints(params);

    // Empty string should be treated as missing, leaving placeholder
    expect(endpoints[0]!.url({ home: "" })).toBe("homes/{home}/sensors");
    // Non-empty value should replace placeholder
    expect(endpoints[0]!.url({ home: "main" })).toBe("homes/main/sensors");
  });
});
