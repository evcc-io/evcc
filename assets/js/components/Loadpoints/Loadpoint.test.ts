import { enableAutoUnmount, shallowMount } from "@vue/test-utils";
import { afterEach, describe, expect, test } from "vite-plus/test";
import { PHASES, type Vehicle } from "@/types/evcc";
import Loadpoint from "./Loadpoint.vue";
import Phases from "./Phases.vue";

enableAutoUnmount(afterEach);

const vehicle: Vehicle = {
  name: "e-up",
  title: "e-up",
  phases: PHASES.TWO_PHASES,
  repeatingPlans: null,
  planStrategy: { continuous: false, precondition: 0 },
};

function render(props: Partial<InstanceType<typeof Loadpoint>["$props"]> = {}) {
  return shallowMount(Loadpoint, {
    props: {
      id: "1",
      socPerKwh: 1,
      maxCurrent: 32,
      phasesConfigured: 0,
      phasesActive: 1,
      offeredCurrent: 16,
      chargeCurrents: [16, 0.1, 0],
      ...props,
    },
    global: {
      stubs: { Phases: false },
      mocks: {
        $t: (key: string) => key,
        $i18n: { locale: "en" },
      },
      directives: { tooltip: {} },
    },
  });
}

describe("phase indicator limits", () => {
  test("fills the 16A vehicle indicator on a 32A charger and hides the third phase", () => {
    const wrapper = render({
      vehicles: [vehicle],
      vehicleName: vehicle.name,
      effectiveMaxCurrent: 16,
    });
    const phases = wrapper.findComponent(Phases);

    expect(phases.props("maxCurrent")).toBe(16);
    expect(phases.props("phasesConfigured")).toBe(2);
    expect(phases.find(".target").attributes("style")).toBe("width: 100%;");
    expect(phases.find(".real").attributes("style")).toBe("width: 100%;");
    expect(phases.findAll(".phase-hidden")).toHaveLength(1);
  });

  test("scales partial current against the effective vehicle maximum", () => {
    const phases = render({
      vehicles: [vehicle],
      vehicleName: vehicle.name,
      effectiveMaxCurrent: 16,
      offeredCurrent: 8,
      chargeCurrents: [8, 0, 0],
    }).findComponent(Phases);

    expect(phases.find(".target").attributes("style")).toBe("width: 50%;");
    expect(phases.find(".real").attributes("style")).toBe("width: 50%;");
  });

  test.each([undefined, 0])(
    "falls back to loadpoint limits with effective current %s",
    (current) => {
      const phases = render({ effectiveMaxCurrent: current }).findComponent(Phases);

      expect(phases.props("maxCurrent")).toBe(32);
      expect(phases.props("phasesConfigured")).toBe(3);
      expect(phases.find(".real").attributes("style")).toBe("width: 50%;");
      expect(phases.findAll(".phase-hidden")).toHaveLength(0);
    }
  );

  test.each([undefined, PHASES.AUTO])(
    "falls back to charger phases when vehicle phases are %s",
    (phases) => {
      const indicator = render({
        vehicles: [{ ...vehicle, phases }],
        vehicleName: vehicle.name,
        effectiveMaxCurrent: 16,
      }).findComponent(Phases);

      expect(indicator.props("phasesConfigured")).toBe(3);
      expect(indicator.props("maxCurrent")).toBe(16);
    }
  );

  test("falls back to charger phases when the selected vehicle is unavailable", () => {
    const phases = render({
      vehicles: [vehicle],
      vehicleName: "unknown",
    }).findComponent(Phases);

    expect(phases.props("phasesConfigured")).toBe(3);
    expect(phases.props("maxCurrent")).toBe(32);
  });

  test("keeps the charger's lower current and phase limits", () => {
    const phases = render({
      vehicles: [vehicle],
      vehicleName: vehicle.name,
      maxCurrent: 10,
      effectiveMaxCurrent: 10,
      phasesConfigured: 1,
      offeredCurrent: 10,
      chargeCurrents: [10, 0, 0],
    }).findComponent(Phases);

    expect(phases.props("maxCurrent")).toBe(10);
    expect(phases.props("phasesConfigured")).toBe(1);
    expect(phases.find(".real").attributes("style")).toBe("width: 100%;");
    expect(phases.findAll(".phase-hidden")).toHaveLength(2);
  });

  test("respects a physically single-phase charger in automatic mode", () => {
    const phases = render({
      vehicles: [vehicle],
      vehicleName: vehicle.name,
      chargerSinglePhase: true,
    }).findComponent(Phases);

    expect(phases.props("phasesConfigured")).toBe(1);
    expect(phases.findAll(".phase-hidden")).toHaveLength(2);
  });

  test("preserves measured phases and mismatch warnings beyond the vehicle limit", () => {
    const phases = render({
      vehicles: [vehicle],
      vehicleName: vehicle.name,
      effectiveMaxCurrent: 16,
      chargeCurrents: [16, 16, 16],
    }).findComponent(Phases);

    expect(phases.classes()).toContain("phases-warning");
    expect(phases.findAll(".phase-inactive")).toHaveLength(0);
    expect(phases.findAll(".phase-hidden")).toHaveLength(0);
  });

  test("uses the vehicle phase limit when active phases and currents are unknown", () => {
    const phases = render({
      vehicles: [vehicle],
      vehicleName: vehicle.name,
      phasesActive: 0,
      chargeCurrents: undefined,
      effectiveMaxCurrent: 16,
    }).findComponent(Phases);

    expect(phases.findAll(".phase-inactive")).toHaveLength(1);
    expect(phases.findAll(".phase-hidden")).toHaveLength(1);
    expect(phases.classes()).not.toContain("phases-warning");
  });

  test("updates limits when switching vehicles and disconnecting", async () => {
    const wrapper = render({
      vehicles: [vehicle, { ...vehicle, name: "three-phase", phases: PHASES.THREE_PHASES }],
      vehicleName: vehicle.name,
      effectiveMaxCurrent: 16,
    });
    const phases = wrapper.findComponent(Phases);

    expect(phases.props("maxCurrent")).toBe(16);
    expect(phases.props("phasesConfigured")).toBe(2);

    await wrapper.setProps({ vehicleName: "three-phase", effectiveMaxCurrent: 32 });
    expect(phases.props("maxCurrent")).toBe(32);
    expect(phases.props("phasesConfigured")).toBe(3);

    await wrapper.setProps({ vehicleName: "", effectiveMaxCurrent: undefined });
    expect(phases.props("maxCurrent")).toBe(32);
    expect(phases.props("phasesConfigured")).toBe(3);
  });
});
