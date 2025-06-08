import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import Status from "./Status.vue";
import { CURRENCY } from "@/types/evcc";

const serializeData = (data: Object) => (data ? `:${JSON.stringify(data)}` : "");
config.global.mocks["$t"] = (key: string, data: Object) => `${key}${serializeData(data)}`;
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

const expectEntries = (props: InstanceType<typeof Status>["$props"], entries: Object) => {
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
		expectEntries({ connected: false }, { charger: "main.vehicleStatus.disconnected" });
	});
	test("vehicle is connected", () => {
		expectEntries({ connected: true }, { charger: "main.vehicleStatus.connected" });
	});
	test("show waiting for vehicle if charger is enabled but not charging", () => {
		expectEntries(
			{ enabled: true, connected: true },
			{ charger: "main.vehicleStatus.waitForVehicle" }
		);
	});
	test("vehicle is charging", () => {
		expectEntries(
			{ connected: true, charging: true },
			{ charger: "main.vehicleStatus.charging" }
		);
	});
});

describe("min charge", () => {
	test("active when vehicle soc is below", () => {
		expectEntries(
			{ connected: true, minSoc: 20, vehicleSoc: 10 },
			{ charger: "main.vehicleStatus.connected", minsoc: "20 %" }
		);
	});
	test("not active when vehicle soc is above", () => {
		expectEntries(
			{ connected: true, minSoc: 20, vehicleSoc: 21 },
			{ charger: "main.vehicleStatus.connected", minsoc: false }
		);
	});
	test("not active when vehicle soc is equal", () => {
		expectEntries(
			{ connected: true, minSoc: 20, vehicleSoc: 20 },
			{ charger: "main.vehicleStatus.connected", minsoc: false }
		);
	});
	test("not active when limit is 0", () => {
		expectEntries(
			{ connected: true, minSoc: 0, vehicleSoc: 10 },
			{ charger: "main.vehicleStatus.connected", minsoc: false }
		);
	});
});

describe("plan", () => {
	const effectivePlanTime = "2020-03-16T06:00:00Z";
	const planProjectedStart = "2020-03-16T02:00:00Z";
	const planProjectedEnd = "2020-03-16T05:00:00Z";
	test("charging if target time is set, status is charging but planned slot is not active", () => {
		expectEntries(
			{ effectivePlanTime, charging: true, connected: true },
			{ charger: "main.vehicleStatus.charging" }
		);
	});
	test("active if target time is set, status is charging and planned slot is active", () => {
		expectEntries(
			{ planProjectedEnd, planActive: true, charging: true, connected: true },
			{ charger: "main.vehicleStatus.charging", planactive: "Mo 06:00" }
		);
	});
	test("waiting for vehicle if a target time is set, the charger is enabled but not charging", () => {
		expectEntries(
			{ planProjectedEnd, planActive: true, enabled: true, connected: true },
			{ charger: "main.vehicleStatus.waitForVehicle", planactive: "Mo 06:00" }
		);
	});
	test("show projected start if not enabled yet", () => {
		expectEntries(
			{ effectivePlanTime, planProjectedStart, connected: true },
			{ charger: "main.vehicleStatus.connected", planstart: "Mo 03:00" }
		);
	});
	test("dont show plan status if plan is disabled (e.g. off, fast mode)", () => {
		expectEntries(
			{
				effectivePlanTime,
				planActive: true,
				charging: true,
				connected: true,
				chargingPlanDisabled: true,
			},
			{ charger: "main.vehicleStatus.charging" }
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
			{ charger: "main.vehicleStatus.waitForVehicle" }
		);
		expectEntries(
			{ effectivePlanTime, planProjectedStart, connected: true, chargingPlanDisabled: true },
			{ charger: "main.vehicleStatus.connected" }
		);
	});
});

describe("climating", () => {
	test("show climating status", () => {
		expectEntries(
			{ connected: true, enabled: true, vehicleClimaterActive: true, charging: true },
			{ charger: "main.vehicleStatus.charging", climater: true }
		);
		expectEntries(
			{ connected: true, enabled: true, vehicleClimaterActive: true, charging: false },
			{ charger: "main.vehicleStatus.waitForVehicle", climater: true }
		);
	});
	test("only show climating if enabled", () => {
		expectEntries(
			{ connected: true, enabled: false, vehicleClimaterActive: true, charging: false },
			{ charger: "main.vehicleStatus.connected", climater: true }
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
			{ charger: "main.vehicleStatus.connected", pvtimer: "1:30\u202Fm" }
		);
	});
	test("don't show pv enable timer if value is zero", () => {
		expectEntries(
			{
				pvAction: "enable",
				connected: true,
				pvRemainingInterpolated: 0,
			},
			{ charger: "main.vehicleStatus.connected" }
		);
	});
	test("show pv disable timer if charging and timer exists", () => {
		expectEntries(
			{
				pvAction: "disable",
				connected: true,
				charging: true,
				pvRemainingInterpolated: 90,
			},
			{ charger: "main.vehicleStatus.charging", pvtimer: "1:30\u202Fm" }
		);
	});
	test("show phase enable timer if it exists", () => {
		expectEntries(
			{
				phaseAction: "scale1p",
				connected: true,
				charging: true,
				phaseRemainingInterpolated: 91,
			},
			{ charger: "main.vehicleStatus.charging", phasetimer: "1:31\u202Fm" }
		);
	});
	test("show phase disable timer if it exists", () => {
		expectEntries(
			{
				phaseAction: "scale3p",
				connected: true,
				charging: true,
				phaseRemainingInterpolated: 91,
			},
			{ charger: "main.vehicleStatus.charging", phasetimer: "1:31\u202Fm" }
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
			{ charger: "main.vehicleStatus.finished", limit: "70 %" }
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
			{ charger: "main.vehicleStatus.finished", limit: "70 %" }
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
			{ charger: "main.vehicleStatus.charging", smartcost: "≤ 500 g" }
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
			{ charger: "main.vehicleStatus.charging", smartcost: "400 g ≤ 500 g" }
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
			{ charger: "main.vehicleStatus.charging", smartcost: "≤ 29,0 ct" }
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
			{ charger: "main.vehicleStatus.charging", smartcost: "28,0 ct ≤ 29,0 ct" }
		);
	});
});
