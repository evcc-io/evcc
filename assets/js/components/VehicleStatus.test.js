import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import VehicleStatus from "./VehicleStatus.vue";

const serializeData = (data) => (data ? `:${JSON.stringify(data)}` : "");
config.global.mocks["$t"] = (key, data) => `${key}${serializeData(data)}`;
config.global.mocks["$i18n"] = { locale: "de-DE" };

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
  test("show waiting for vehicle if charger is enabled but not charging", () => {
    expectStatus({ enabled: true, connected: true }, "waitForVehicle");
  });
  test("vehicle is charging", () => {
    expectStatus({ connected: true, charging: true }, "charging");
  });
});

describe("min charge", () => {
  test("active when vehicle soc is below", () => {
    expectStatus({ connected: true, minSoc: 20, vehicleSoc: 10 }, "minCharge", { soc: 20 });
  });
  test("not active when vehicle soc is above", () => {
    expectStatus({ connected: true, minSoc: 20, vehicleSoc: 21 }, "connected");
  });
  test("not active when vehicle soc is equal", () => {
    expectStatus({ connected: true, minSoc: 20, vehicleSoc: 20 }, "connected");
  });
  test("not active when limit is 0", () => {
    expectStatus({ connected: true, minSoc: 0, vehicleSoc: 10 }, "connected");
  });
});

describe("plan", () => {
  const targetTime = "2020-03-16T06:00:00Z";
  const planProjectedStart = "2020-03-16T02:00:00Z";
  test("charging if target time is set, status is charging but planned slot is not active", () => {
    expectStatus({ targetTime, charging: true, connected: true }, "charging");
  });
  test("active if target time is set, status is charging and planned slot is active", () => {
    expectStatus(
      { targetTime, planActive: true, charging: true, connected: true },
      "targetChargeActive"
    );
  });
  test("waiting for vehicle if a target time is set, the charger is enabled but not charging", () => {
    expectStatus(
      { targetTime, planActive: true, enabled: true, connected: true },
      "targetChargeWaitForVehicle"
    );
  });
  test("show projected start if not enabled yet", () => {
    expectStatus({ targetTime, planProjectedStart, connected: true }, "targetChargePlanned", {
      time: "Mo 03:00",
    });
  });
});

describe("climating", () => {
  test("show climating status", () => {
    expectStatus(
      { connected: true, enabled: true, climaterActive: true, charging: true },
      "climating"
    );
    expectStatus(
      { connected: true, enabled: true, climaterActive: true, charging: false },
      "climating"
    );
  });
  test("only show climating if enabled", () => {
    expectStatus({ connected: true, enabled: false, climaterActive: true }, "connected");
  });
});

describe("timer", () => {
  test("show pv enable timer if not enabled yet and timer exists", () => {
    expectStatus(
      {
        pvAction: "enable",
        connected: true,
        pvRemainingInterpolated: 90,
      },
      "pvEnable",
      { remaining: "1:30\u202Fm" }
    );
  });
  test("don't show pv enable timer if value is zero", () => {
    expectStatus(
      {
        pvAction: "enable",
        connected: true,
        pvRemainingInterpolated: 0,
      },
      "connected"
    );
  });
  test("show pv disable timer if charging and timer exists", () => {
    expectStatus(
      {
        pvAction: "disable",
        connected: true,
        charging: true,
        pvRemainingInterpolated: 90,
      },
      "pvDisable",
      { remaining: "1:30\u202Fm" }
    );
  });
  test("show phase enable timer if it exists", () => {
    expectStatus(
      {
        phaseAction: "scale1p",
        connected: true,
        charging: true,
        phaseRemainingInterpolated: 90,
      },
      "scale1p",
      { remaining: "1:30\u202Fm" }
    );
  });
  test("show phase disable timer if it exists", () => {
    expectStatus(
      {
        phaseAction: "scale3p",
        connected: true,
        charging: true,
        phaseRemainingInterpolated: 90,
      },
      "scale3p",
      { remaining: "1:30\u202Fm" }
    );
  });
  test("show guard timer if it exists", () => {
    expectStatus(
      {
        connected: true,
        guardAction: "enable",
        guardRemainingInterpolated: 90,
      },
      "guard",
      { remaining: "1:30\u202Fm" }
    );
  });
  test("don't show guard timer if another timer exists", () => {
    expectStatus(
      {
        connected: true,
        charging: true,
        pvAction: "disable",
        pvRemainingInterpolated: 30,
        guardAction: "enable",
        guardRemainingInterpolated: 90,
      },
      "pvDisable",
      { remaining: "30\u202Fs" }
    );
  });
  test("show guard timer if charging and no other timer exists", () => {
    expectStatus(
      {
        connected: true,
        charging: true,
        guardAction: "enable",
        guardRemainingInterpolated: 90,
      },
      "guard",
      { remaining: "1:30\u202Fm" }
    );
  });
});

describe("vehicle target soc", () => {
  test("show target reached if charger enabled but soc has reached vehicle limit", () => {
    expectStatus(
      {
        connected: true,
        enabled: true,
        vehicleTargetSoc: 70,
        vehicleSoc: 70,
      },
      "vehicleTargetReached",
      { soc: 70 }
    );
  });
  test("show reached message even if vehicle is slightly below its limit", () => {
    expectStatus(
      {
        connected: true,
        enabled: true,
        vehicleTargetSoc: 70,
        vehicleSoc: 69,
      },
      "vehicleTargetReached",
      { soc: 70 }
    );
  });
});

describe("smart grid charging", () => {
  test("show clean energy message", () => {
    expectStatus(
      {
        connected: true,
        enabled: true,
        charging: true,
        tariffCo2: 400,
        smartCostLimit: 500,
        smartCostType: "co2",
        smartCostActive: true,
      },
      `cleanEnergyCharging:{"co2":"400 g","limit":"500 g"}`
    );
  });
  test("show cheap energy message", () => {
    expectStatus(
      {
        connected: true,
        enabled: true,
        charging: true,
        tariffGrid: 0.28,
        smartCostLimit: 0.29,
        currency: "CHF",
        smartCostActive: true,
      },
      `cheapEnergyCharging:{"price":"28,0 rp","limit":"29,0 rp"}`
    );
  });
});
