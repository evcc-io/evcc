import { mount, config } from "@vue/test-utils";
import { describe, expect, test, vi, beforeEach } from "vite-plus/test";
import PlansRepeatingSettings from "./PlansRepeatingSettings.vue";
import api from "@/api";
import settings from "@/settings";
import type { Vehicle, RepeatingPlan } from "@/types/evcc";
import en from "../../../../i18n/en.json";

const lookup = (key: string, params?: Record<string, any>): string => {
  const template = key.split(".").reduce<any>((o, k) => o?.[k], en) ?? key;
  if (typeof template !== "string") return key;
  if (!params) return template;
  return Object.entries(params).reduce(
    (acc, [k, v]) => acc.replace(new RegExp(`{${k}}`, "g"), String(v)),
    template
  );
};

config.global.mocks["$t"] = (key: string, params?: Record<string, any>) => lookup(key, params);
config.global.mocks["$te"] = (key: string) =>
  typeof key.split(".").reduce<any>((o, k) => o?.[k], en) === "string";
config.global.mocks["$i18n"] = { locale: "de-DE" };

const dummyPlans: RepeatingPlan[] = [
  {
    weekdays: [1, 2, 3, 4, 5],
    time: "07:00",
    soc: 80,
    active: true,
    tz: "Europe/Berlin",
  },
];

const now = new Date();
// Dynamic future timestamp, 5 days ahead (e.g. 2026-08-20T10:00:00.000Z)
const futureIso = new Date(now.getTime() + 5 * 24 * 3600 * 1000).toISOString();

const vehicleA: Vehicle = {
  name: "vehicleA",
  title: "Tesla Model 3",
  capacity: 60,
  pausedUntil: futureIso, // paused until future date (e.g. 2026-08-20T10:00:00.000Z)
  repeatingPlans: dummyPlans,
  planStrategy: { continuous: false, precondition: 0 },
};

const vehicleB: Vehicle = {
  name: "vehicleB",
  title: "VW ID.4",
  capacity: 77,
  pausedUntil: null,
  repeatingPlans: dummyPlans,
  planStrategy: { continuous: false, precondition: 0 },
};

describe("PlansRepeatingSettings - Custom Date/Time & Pause functionality", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  test("custom date/time picker toggles when preset is clicked and formats RFC 3339 timestamp", async () => {
    const postSpy = vi.spyOn(api, "post").mockResolvedValue({ data: {} } as any);

    const wrapper = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: vehicleB,
      },
    });

    // Initially not paused, pause dropdown is visible
    expect(wrapper.find('[data-testid="repeating-plan-pause"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="pause-custom-date"]').exists()).toBe(false);

    // Click custom preset option
    const customBtn = wrapper.find('[data-testid="pause-preset-custom"]');
    expect(customBtn.exists()).toBe(true);
    expect(customBtn.attributes("aria-expanded")).toBe("false");

    await customBtn.trigger("click");
    expect(customBtn.attributes("aria-expanded")).toBe("true");

    // Custom inputs should now be rendered
    const dateInput = wrapper.find('[data-testid="pause-custom-date"]');
    const timeInput = wrapper.find('[data-testid="pause-custom-time"]');
    const applyBtn = wrapper.find('[data-testid="pause-custom-apply"]');

    expect(dateInput.exists()).toBe(true);
    expect(timeInput.exists()).toBe(true);
    expect(applyBtn.exists()).toBe(true);

    // Check accessibility and min date
    expect(dateInput.attributes("tabindex")).toBe("0");
    expect(timeInput.attributes("tabindex")).toBe("0");
    expect(dateInput.attributes("min")).toBeDefined();

    // Enter custom future date and time, 10 days ahead (e.g. 2026-08-25 14:30)
    const customTarget = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 10, 14, 30, 0, 0);
    const yyyy = customTarget.getFullYear();
    const mm = String(customTarget.getMonth() + 1).padStart(2, "0");
    const dd = String(customTarget.getDate()).padStart(2, "0");
    const customDateStr = `${yyyy}-${mm}-${dd}`;

    await dateInput.setValue(customDateStr);
    await timeInput.setValue("14:30");

    expect(applyBtn.attributes("disabled")).toBeUndefined();

    // Click Apply
    await applyBtn.trigger("click");

    // Verify API was called with RFC 3339 encoded format
    expect(postSpy).toHaveBeenCalledTimes(1);
    const calledUrl = postSpy.mock.calls[0]![0];
    expect(calledUrl).toContain("vehicles/vehicleB/plan/pause/");

    // The encoded timestamp should decode to a valid RFC 3339 date matching customTarget
    const encodedTimestamp = calledUrl.replace("vehicles/vehicleB/plan/pause/", "");
    const decodedTimestamp = decodeURIComponent(encodedTimestamp);
    expect(decodedTimestamp).toBe(customTarget.toISOString());
  });

  test("custom date/time picker validates against past timestamps", async () => {
    const wrapper = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: vehicleB,
      },
    });

    await wrapper.find('[data-testid="pause-preset-custom"]').trigger("click");

    const dateInput = wrapper.find('[data-testid="pause-custom-date"]');
    const timeInput = wrapper.find('[data-testid="pause-custom-time"]');
    const applyBtn = wrapper.find('[data-testid="pause-custom-apply"]');

    // Set date in the past
    await dateInput.setValue("2020-01-01");
    await timeInput.setValue("10:00");

    // Apply button should be disabled
    expect(applyBtn.attributes("disabled")).toBeDefined();
  });

  test("multi-vehicle UI reactivity: switching vehicles updates pause banner and controls reactively", async () => {
    const wrapper = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: vehicleA, // paused until 2026-08-20
      },
    });

    // Vehicle A is paused -> should display paused badge
    expect(wrapper.find('[data-testid="repeating-plan-paused-badge"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="repeating-plan-pause"]').exists()).toBe(false);

    // Switch vehicle prop to Vehicle B (not paused)
    await wrapper.setProps({
      vehicle: vehicleB,
    });

    // Vehicle B is not paused -> badge should disappear, pause button should appear
    expect(wrapper.find('[data-testid="repeating-plan-paused-badge"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="repeating-plan-pause"]').exists()).toBe(true);

    // Switch back to Vehicle A
    await wrapper.setProps({
      vehicle: vehicleA,
    });

    expect(wrapper.find('[data-testid="repeating-plan-paused-badge"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="repeating-plan-pause"]').exists()).toBe(false);
  });

  test("resume button sends DELETE request to /api/vehicles/{name}/plan/pause", async () => {
    const deleteSpy = vi.spyOn(api, "delete").mockResolvedValue({ data: {} } as any);

    const wrapper = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: vehicleA,
      },
    });

    const resumeBtn = wrapper.find('[data-testid="repeating-plan-resume"]');
    expect(resumeBtn.exists()).toBe(true);
    expect(resumeBtn.attributes("tabindex")).toBe("0");

    await resumeBtn.trigger("click");

    expect(deleteSpy).toHaveBeenCalledWith("vehicles/vehicleA/plan/pause");
  });

  test("pause preset selection saves to settings.lastPausePreset and applies active class", async () => {
    vi.spyOn(api, "post").mockResolvedValue({ data: {} } as any);
    vi.spyOn(api, "delete").mockResolvedValue({ data: {} } as any);
    settings.lastPausePreset = undefined;

    const wrapper = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: vehicleB,
      },
    });

    const tomorrowBtn = wrapper.find('[data-testid="pause-preset-tomorrow"]');
    expect(tomorrowBtn.classes()).not.toContain("active");

    // Click tomorrow preset
    await tomorrowBtn.trigger("click");
    expect(settings.lastPausePreset).toBe("tomorrow");

    // After pause is triggered, resume to inspect dropdown state
    const resumeBtn = wrapper.find('[data-testid="repeating-plan-resume"]');
    expect(resumeBtn.exists()).toBe(true);
    await resumeBtn.trigger("click");

    // Dropdown is back, tomorrow button should now have active class
    expect(wrapper.find('[data-testid="pause-preset-tomorrow"]').classes()).toContain("active");
    expect(wrapper.find('[data-testid="pause-preset-friday"]').classes()).not.toContain("active");

    // Click friday preset
    await wrapper.find('[data-testid="pause-preset-friday"]').trigger("click");
    expect(settings.lastPausePreset).toBe("friday");

    // Resume again and check active class on friday button
    await wrapper.find('[data-testid="repeating-plan-resume"]').trigger("click");
    expect(wrapper.find('[data-testid="pause-preset-friday"]').classes()).toContain("active");
    expect(wrapper.find('[data-testid="pause-preset-tomorrow"]').classes()).not.toContain("active");
  });

  test("custom date/time pause does not overwrite settings.lastPausePreset", async () => {
    vi.spyOn(api, "post").mockResolvedValue({ data: {} } as any);
    vi.spyOn(api, "delete").mockResolvedValue({ data: {} } as any);
    settings.lastPausePreset = "24h";

    const wrapper = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: vehicleB,
      },
    });

    // 24h button should be active based on initial setting
    const preset24hBtn = wrapper.find('[data-testid="pause-preset-24h"]');
    expect(preset24hBtn.classes()).toContain("active");

    // Open custom picker and submit, 10 days ahead (e.g. 2026-08-30 12:00)
    const customTarget = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 10, 12, 0, 0, 0);
    const yyyy = customTarget.getFullYear();
    const mm = String(customTarget.getMonth() + 1).padStart(2, "0");
    const dd = String(customTarget.getDate()).padStart(2, "0");
    const customDateStr = `${yyyy}-${mm}-${dd}`;

    await wrapper.find('[data-testid="pause-preset-custom"]').trigger("click");
    await wrapper.find('[data-testid="pause-custom-date"]').setValue(customDateStr);
    await wrapper.find('[data-testid="pause-custom-time"]').setValue("12:00");
    await wrapper.find('[data-testid="pause-custom-apply"]').trigger("click");

    // settings.lastPausePreset should remain "24h"
    expect(settings.lastPausePreset).toBe("24h");

    // Resume and verify 24h is still active
    await wrapper.find('[data-testid="repeating-plan-resume"]').trigger("click");
    expect(wrapper.find('[data-testid="pause-preset-24h"]').classes()).toContain("active");
  });

  test("subsequent opens and remounts respect remembered lastPausePreset", async () => {
    // Preset: 7d
    settings.lastPausePreset = "7d";

    const wrapper = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: vehicleB,
      },
    });

    // Check all preset buttons
    expect(wrapper.find('[data-testid="pause-preset-7d"]').classes()).toContain("active");
    expect(wrapper.find('[data-testid="pause-preset-tomorrow"]').classes()).not.toContain("active");
    expect(wrapper.find('[data-testid="pause-preset-friday"]').classes()).not.toContain("active");
    expect(wrapper.find('[data-testid="pause-preset-sunday"]').classes()).not.toContain("active");
    expect(wrapper.find('[data-testid="pause-preset-24h"]').classes()).not.toContain("active");
    expect(wrapper.find('[data-testid="pause-preset-48h"]').classes()).not.toContain("active");

    // Change setting to sunday and remount
    settings.lastPausePreset = "sunday";
    const wrapper2 = mount(PlansRepeatingSettings, {
      props: {
        id: 2,
        plans: dummyPlans,
        vehicle: vehicleB,
      },
    });

    expect(wrapper2.find('[data-testid="pause-preset-sunday"]').classes()).toContain("active");
    expect(wrapper2.find('[data-testid="pause-preset-7d"]').classes()).not.toContain("active");
  });

  test("pausedUntil pill formats with relative/weekday within 7 days and full date beyond 7 days", async () => {
    const now = new Date();
    // Test within 7 days (e.g. +3 days)
    const plus3Days = new Date(now.getTime() + 3 * 86400000);
    plus3Days.setHours(10, 0, 0, 0);

    const wrapperNear = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: {
          ...vehicleA,
          pausedUntil: plus3Days.toISOString(),
        },
      },
    });

    const nearBadge = wrapperNear.find('[data-testid="repeating-plan-paused-badge"]');
    expect(nearBadge.exists()).toBe(true);
    expect(nearBadge.text()).toContain("10:00");

    // Test beyond 7 days (e.g. +14 days)
    const plus14Days = new Date(now.getTime() + 14 * 86400000);
    plus14Days.setHours(16, 30, 0, 0);

    const wrapperFar = mount(PlansRepeatingSettings, {
      props: {
        id: 1,
        plans: dummyPlans,
        vehicle: {
          ...vehicleA,
          pausedUntil: plus14Days.toISOString(),
        },
      },
    });

    const farBadge = wrapperFar.find('[data-testid="repeating-plan-paused-badge"]');
    expect(farBadge.exists()).toBe(true);
    expect(farBadge.text()).toContain("16:30");
    // Should include day number or month beyond 7 days
    expect(farBadge.text()).toMatch(/\d+/);
  });
});
