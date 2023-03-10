import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import VehicleStatus from "./VehicleStatus.vue";

const serializeData = (data) => (data ? `:${JSON.stringify(data)}` : "");
config.global.mocks["$t"] = (key, data) => `${key}${serializeData(data)}`;

const expectStatus = (props, messageKey, data) => {
  const wrapper = mount(VehicleStatus, { props });
  expect(wrapper.find("div").text()).eq(`main.vehicleStatus.${messageKey}${serializeData(data)}`);
};

describe("basics", () => {
  test("no vehicle is connected", () => {
    expectStatus({ connected: false }, "disconnected");
  });
  test("vehicle is connected", () => {
    expectStatus({ connected: true }, "connected");
  });
  test("min charge active", () => {
    expectStatus({ connected: true, minSoc: 20, vehicleSoc: 10 }, "minCharge", { soc: 20 });
  });
});
