import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import Status from "./Status.vue";
import { CURRENCY } from "@/types/evcc";
import en from "../../../../i18n/en.json";

// minimal $t/$te that walk en.json so tests assert on real English text
const lookup = (key: string): string | undefined => {
  const v = key.split(".").reduce<any>((o, k) => o?.[k], en);
  return typeof v === "string" ? v : undefined;
};
config.global.mocks["$t"] = (key: string) => lookup(key) ?? key;
config.global.mocks["$te"] = (key: string) => lookup(key) !== undefined;
config.global.mocks["$i18n"] = { locale: "de-DE" };

const allEntries = {
  pvtimer: false,
  phasetimer: false,
  solar: false,
  climater: false,
  minsoc: false,
  limit: false,
  smartcost: false,
  planactive: false,
  planstart: false,
};

const expectEntries = (props: InstanceType<typeof Status>["$props"], entries: object) => {
  const expectedEntries = { ...allEntries, ...entries };

  const wrapper = mount(Status, { props });

  Object.entries(expectedEntries).forEach(([key, value]) => {
    const selector = `[data-testid=vehicle-status-${key}]`;
    if (typeof value === "boolean") {
      expect(wrapper.find(selector).exists(), selector).eq(value);
    } else {
      expect(wrapper.find(selector).exists(), selector).eq(true);
      expect(wrapper.find(selector).text(), selector).eq(value);
    }
  });
};

describe("basics", () => {
  test("no vehicle is connected", () => {
    expectEntries({ connected: false }, { charger: "Disconnected." });
  });
  test("vehicle is connected", () => {
    expectEntries({ connected: true }, { charger: "Connected." });
  });
  test("show waiting for vehicle if charger is enabled but not charging", () => {
    expectEntries({ enabled: true, connected: true }, { charger: "Ready. Waiting for vehicle…" });
  });
  test("show waiting for authorization if enabled but charger requires auth", () => {
    expectEntries(
      { enabled: true, connected: true, chargerStatusReason: "waitingforauthorization" },
      { charger: "Connected. Waiting for authorization…" }
    );
  });
  test("vehicle is charging", () => {
    expectEntries({ connected: true, enabled: true, charging: true }, { charger: "Charging…" });
  });
});

describe("min charge", () => {
  test("active when minSocNotReached is true", () => {
    expectEntries(
      { connected: true, minSoc: 20, minSocNotReached: true },
      { charger: "Connected.", minsoc: "20 %" }
    );
  });
  test("not active when minSocNotReached is false", () => {
    expectEntries(
      { connected: true, minSoc: 20, minSocNotReached: false },
      { charger: "Connected.", minsoc: false }
    );
  });
  test("not active when minSocNotReached is not set", () => {
    expectEntries({ connected: true, minSoc: 20 }, { charger: "Connected.", minsoc: false });
  });
});

describe("plan", () => {
  const effectivePlanTime = "2020-03-16T06:00:00Z";
  const planProjectedStart = "2020-03-16T02:00:00Z";
  const planProjectedEnd = "2020-03-16T05:00:00Z";
  test("charging if target time is set, status is charging but planned slot is not active", () => {
    expectEntries(
      { effectivePlanTime, enabled: true, charging: true, connected: true },
      { charger: "Charging…" }
    );
  });
  test("active if target time is set, status is charging and planned slot is active", () => {
    expectEntries(
      { planProjectedEnd, planActive: true, enabled: true, charging: true, connected: true },
      { charger: "Charging…", planactive: "Mo 06:00" }
    );
  });
  test("waiting for vehicle if a target time is set, the charger is enabled but not charging", () => {
    expectEntries(
      { planProjectedEnd, planActive: true, enabled: true, connected: true },
      { charger: "Ready. Waiting for vehicle…", planactive: "Mo 06:00" }
    );
  });
  test("show projected start if not enabled yet", () => {
    expectEntries(
      { effectivePlanTime, planProjectedStart, connected: true },
      { charger: "Connected.", planstart: "Mo 03:00" }
    );
  });
  test("dont show plan status if plan is disabled (e.g. off, fast mode)", () => {
    expectEntries(
      {
        effectivePlanTime,
        planActive: true,
        enabled: true,
        charging: true,
        connected: true,
        chargingPlanDisabled: true,
      },
      { charger: "Charging…" }
    );
    expectEntries(
      {
        effectivePlanTime,
        planActive: true,
        charging: false,
        enabled: true,
        connected: true,
        chargingPlanDisabled: true,
      },
      { charger: "Ready. Waiting for vehicle…" }
    );
    expectEntries(
      { effectivePlanTime, planProjectedStart, connected: true, chargingPlanDisabled: true },
      { charger: "Connected." }
    );
  });
});

describe("climating", () => {
  test("show climating status", () => {
    expectEntries(
      { connected: true, enabled: true, vehicleClimaterActive: true, charging: true },
      { charger: "Charging…", climater: true }
    );
    expectEntries(
      { connected: true, enabled: true, vehicleClimaterActive: true, charging: false },
      { charger: "Ready. Waiting for vehicle…", climater: true }
    );
  });
  test("only show climating if enabled", () => {
    expectEntries(
      { connected: true, enabled: false, vehicleClimaterActive: true, charging: false },
      { charger: "Connected.", climater: true }
    );
  });
});

describe("timer", () => {
  test("show pv enable timer if not enabled yet and timer exists", () => {
    expectEntries(
      {
        pvAction: "enable",
        connected: true,
        pvRemainingInterpolated: 90,
      },
      { charger: "Connected.", pvtimer: "1:30\u202Fm" }
    );
  });
  test("don't show pv enable timer if value is zero", () => {
    expectEntries(
      {
        pvAction: "enable",
        connected: true,
        pvRemainingInterpolated: 0,
      },
      { charger: "Connected." }
    );
  });
  test("show pv disable timer if charging and timer exists", () => {
    expectEntries(
      {
        pvAction: "disable",
        connected: true,
        enabled: true,
        charging: true,
        pvRemainingInterpolated: 90,
      },
      { charger: "Charging…", pvtimer: "1:30\u202Fm" }
    );
  });
  test("show phase enable timer if it exists", () => {
    expectEntries(
      {
        phaseAction: "scale1p",
        connected: true,
        enabled: true,
        charging: true,
        phaseRemainingInterpolated: 91,
      },
      { charger: "Charging…", phasetimer: "1:31\u202Fm" }
    );
  });
  test("show phase disable timer if it exists", () => {
    expectEntries(
      {
        phaseAction: "scale3p",
        connected: true,
        enabled: true,
        charging: true,
        phaseRemainingInterpolated: 91,
      },
      { charger: "Charging…", phasetimer: "1:31\u202Fm" }
    );
  });
});

describe("vehicle target soc", () => {
  test("show target reached if charger enabled but soc has reached vehicle limit", () => {
    expectEntries(
      {
        connected: true,
        enabled: true,
        vehicleLimitSoc: 70,
        vehicleSoc: 70,
      },
      { charger: "Finished.", limit: "70 %" }
    );
  });
  test("show reached message even if vehicle is slightly below its limit", () => {
    expectEntries(
      {
        connected: true,
        enabled: true,
        vehicleLimitSoc: 70,
        vehicleSoc: 69,
      },
      { charger: "Finished.", limit: "70 %" }
    );
  });
});

describe("smart grid charging", () => {
  test("show clean energy message", () => {
    expectEntries(
      {
        connected: true,
        enabled: true,
        charging: true,
        tariffCo2: 600,
        smartCostLimit: 500,
        smartCostType: "co2",
      },
      { charger: "Charging…", smartcost: "≤ 500 g" }
    );
  });
  test("show clean energy message if active", () => {
    expectEntries(
      {
        connected: true,
        enabled: true,
        charging: true,
        tariffCo2: 400,
        smartCostLimit: 500,
        smartCostType: "co2",
        smartCostActive: true,
      },
      { charger: "Charging…", smartcost: "400 g ≤ 500 g" }
    );
  });
  test("show cheap energy message", () => {
    expectEntries(
      {
        connected: true,
        enabled: true,
        charging: true,
        tariffGrid: 0.3,
        smartCostLimit: 0.29,
        currency: CURRENCY.EUR,
      },
      { charger: "Charging…", smartcost: "≤ 29,0 ct" }
    );
  });
  test("show cheap energy message if active", () => {
    expectEntries(
      {
        connected: true,
        enabled: true,
        charging: true,
        tariffGrid: 0.28,
        smartCostLimit: 0.29,
        currency: CURRENCY.EUR,
        smartCostActive: true,
      },
      { charger: "Charging…", smartcost: "28,0 ct ≤ 29,0 ct" }
    );
  });
});

describe("heating device", () => {
  test("status text per state", () => {
    const base = { heating: true };
    // offline when not connected
    expectEntries({ ...base, connected: false }, { charger: "Disconnected." });
    // standby when connected but not enabled
    expectEntries({ ...base, connected: true }, { charger: "Standby." });
    // ready to heat when enabled but not yet drawing power
    expectEntries({ ...base, connected: true, enabled: true }, { charger: "Ready to heat…" });
    // heating when enabled and drawing power
    expectEntries(
      { ...base, connected: true, enabled: true, charging: true },
      { charger: "Heating…" }
    );
    // target reached when temp limit hit
    expectEntries(
      { ...base, connected: true, enabled: true, vehicleLimitSoc: 60, vehicleSoc: 60 },
      { charger: "Finished.", limit: true }
    );
  });
});

describe("continuous heating device (heat pump)", () => {
  test("status text per state", () => {
    const base = { heating: true, continuous: true };
    // not enabled, not drawing → normal operation
    expectEntries({ ...base, connected: true }, { charger: "Normal operation." });
    // not enabled but device draws power autonomously → still normal operation
    expectEntries({ ...base, connected: true, charging: true }, { charger: "Normal operation." });
    // enabled, not yet drawing → boost requested
    expectEntries({ ...base, connected: true, enabled: true }, { charger: "Boost requested…" });
    // enabled and drawing → boost active
    expectEntries(
      { ...base, connected: true, enabled: true, charging: true },
      { charger: "Boost active." }
    );
    // limit reached cascades to heatingStatus
    expectEntries(
      { ...base, connected: true, enabled: true, vehicleLimitSoc: 60, vehicleSoc: 60 },
      { charger: "Finished.", limit: true }
    );
  });
});
