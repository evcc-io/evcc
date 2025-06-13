import { CHARGE_MODE } from "@/types/evcc";
import Vehicle from "./Vehicle.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

const baseState = {
	vehicle: {
		title: "Mein Auto",
		icon: "car",
		capacity: 72,
		features: [],
		name: "",
		repeatingPlans: [],
	},
	enabled: false,
	connected: true,
	vehicleName: "meinauto",
	vehicleSoc: 42.742,
	vehicleRange: 231,
	limitSoc: 90,
	chargedEnergy: 14123,
	socBasedCharging: true,
	id: 0,
};

export default {
	title: "Vehicles/Vehicle",
	component: Vehicle,
	argTypes: {
		chargedEnergy: { control: "number" },
		charging: { control: "boolean" },
		vehicleClimaterActive: { control: "boolean" },
		vehicleWelcomeActive: { control: "boolean" },
		connected: { control: "boolean" },
		currency: { control: "text" },
		effectiveLimitSoc: { control: "number" },
		effectivePlanSoc: { control: "number" },
		effectivePlanTime: { control: "text" },
		batteryBoostActive: { control: "boolean" },
		enabled: { control: "boolean" },
		heating: { control: "boolean" },
		id: { control: "text" },
		integratedDevice: { control: "boolean" },
		limitEnergy: { control: "number" },
		mode: { control: "text" },
		chargerStatusReason: { control: "text" },
		vehicle: { control: "object" },
		vehicleName: { control: "text" },
		vehicleRange: { control: "number" },
		vehicleSoc: { control: "number" },
		vehicleLimitSoc: { control: "number" },
		socBasedCharging: { control: "boolean" },
	},
} as Meta<typeof Vehicle>;

const Template: StoryFn<typeof Vehicle> = (args) => ({
	components: { Vehicle },
	setup() {
		return { args };
	},
	template: '<Vehicle v-bind="args" />',
});

export const Disconnected = Template.bind({});
Disconnected.args = {
	...baseState,
	connected: false,
};

export const Connected = Template.bind({});
Connected.args = {
	...baseState,
};

export const Ready = Template.bind({});
Ready.args = {
	...baseState,
	enabled: true,
};

export const Charging = Template.bind({});
Charging.args = {
	...baseState,
	enabled: true,
	charging: true,
};

export const UnknownVehicle = Template.bind({});
UnknownVehicle.args = {
	...baseState,
	enabled: true,
	charging: true,
	vehicleName: "",
	socBasedCharging: false,
	vehicle: { ...baseState.vehicle, capacity: undefined },
	mode: CHARGE_MODE.PV,
};

export const OfflineVehicle = Template.bind({});
OfflineVehicle.args = {
	...baseState,
	enabled: true,
	charging: true,
	socBasedCharging: false,
	vehicle: {
		...baseState.vehicle,
		title: "Opel Corsa-e",
		capacity: 72,
		features: ["Offline"],
	},
	mode: CHARGE_MODE.PV,
};

export const OfflineVehicleWithTarget = Template.bind({});
OfflineVehicleWithTarget.args = {
	...baseState,
	enabled: true,
	charging: true,
	socBasedCharging: false,
	vehicle: {
		...baseState.vehicle,
		title: "Opel Corsa-e",
		capacity: 72,
		features: ["Offline"],
	},
	mode: CHARGE_MODE.PV,
};

export const VehicleLimit = Template.bind({});
VehicleLimit.args = {
	...baseState,
	enabled: true,
	charging: true,
	vehicleLimitSoc: 80,
};

export const VehicleLimitReached = Template.bind({});
VehicleLimitReached.args = {
	...baseState,
	enabled: true,
	vehicleLimitSoc: 80,
	vehicleSoc: 80,
};

export const TargetChargePlanned = Template.bind({});
TargetChargePlanned.args = {
	...baseState,
	mode: CHARGE_MODE.PV,
};

export const TargetChargeActive = Template.bind({});
TargetChargeActive.args = {
	...baseState,
	enabled: true,
	charging: true,
	mode: CHARGE_MODE.PV,
};

export const SmartChargeCostLimitActive = Template.bind({});
SmartChargeCostLimitActive.args = {
	...baseState,
	enabled: true,
	charging: true,
	smartCostLimit: 0.13,
	mode: CHARGE_MODE.PV,
};

export const PvEnableTimer = Template.bind({});
PvEnableTimer.args = {
	...baseState,
	pvAction: "enable",
	pvRemainingInterpolated: 32,
};

export const PvDisableTimer = Template.bind({});
PvDisableTimer.args = {
	...baseState,
	enabled: true,
	charging: true,
	pvAction: "disable",
	pvRemainingInterpolated: 155,
};

export const VehicleSwitch = Template.bind({});
VehicleSwitch.args = {
	...baseState,
	vehicles: ["Blauer e-Golf", "Weißes Model 3"],
};
