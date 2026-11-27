import { describe, expect, test } from "vite-plus/test";
import { prefillFor, remember, sessionDevices, suggestions } from "./onboarding";
import { ConfigType } from "./types/evcc";
import type { Template } from "./components/Config/DeviceModal";

const template: Template = {
  Params: [
    {
      Name: "usage",
      Required: true,
      Advanced: false,
      Deprecated: false,
      Choice: ["pv", "battery", "grid"],
    },
    { Name: "host", Required: true, Advanced: false, Deprecated: false },
    { Name: "capacity", Required: false, Advanced: false, Deprecated: false, Usages: ["battery"] },
    { Name: "maxacpower", Required: false, Advanced: true, Deprecated: false, Usages: ["pv"] },
  ],
  Requirements: { Description: "" },
};

const values = {
  type: ConfigType.Template,
  template: "sma-hybrid",
  deviceTitle: "Home Battery",
  usage: "battery" as const,
  modbus: "tcpip",
  host: "192.168.1.5",
  port: 502,
  id: 3,
  capacity: 10,
};

describe("remember", () => {
  test("keeps connection values, drops usage specific ones, offers other usages", () => {
    remember("meter", "SMA Hybrid", template, values);
    expect(sessionDevices).toHaveLength(1);
    expect(suggestions("pv")).toHaveLength(1);
    expect(suggestions("battery")).toHaveLength(0);
    expect(prefillFor(sessionDevices[0]!, "pv")).toEqual({
      type: ConfigType.Template,
      template: "sma-hybrid",
      usage: "pv",
      modbus: "tcpip",
      host: "192.168.1.5",
      port: 502,
      id: 3,
    });
  });
});
