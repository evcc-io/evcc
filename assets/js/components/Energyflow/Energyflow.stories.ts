import type { Meta, StoryFn } from "@storybook/vue3";
import Energyflow from "./Energyflow.vue";
import { CURRENCY } from "@/types/evcc";

export default {
	title: "Energyflow/Energyflow",
	component: Energyflow,
} as Meta<typeof Energyflow>;

const Template: StoryFn<typeof Energyflow> = (args) => ({
	components: { Energyflow },
	setup() {
		return { args };
	},
	template: '<Energyflow v-bind="args" />',
});

export const GridAndPV = Template.bind({});
GridAndPV.args = {
	gridConfigured: true,
	pvConfigured: true,
	pvPower: 7300,
	gridPower: -2300,
	homePower: 800,
	loadpointsCompact: [
		{
			power: 1000,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 1000,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 1000,
			icon: "bike",
			charging: true,
			title: "Garage",
			chargePower: 1000,
			connected: true,
			index: 1,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 2200,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 2200,
			connected: true,
			index: 2,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
	tariffGrid: 0.25,
	tariffFeedIn: 0.08,
	smartCostType: "price",
	currency: CURRENCY.EUR,
	pv: [{ power: 5000 }, { power: 2300 }],
	forecast: {
		solar: {
			today: { energy: 1000, complete: true },
			tomorrow: { energy: 1000, complete: false },
			dayAfterTomorrow: { energy: 1000, complete: false },
		},
	},
};

export const BatteryAndGrid = Template.bind({});
BatteryAndGrid.args = {
	gridConfigured: true,
	pvConfigured: true,
	batteryConfigured: true,
	pvPower: 0,
	gridPower: 1200,
	homePower: 2000,
	batteryPower: 800,
	batterySoc: 77,
	tariffGrid: 0.25,
	tariffFeedIn: 0.08,
	currency: CURRENCY.EUR,
	battery: [
		{ soc: 44.999, capacity: 13.3, power: 350, controllable: true },
		{ soc: 82.3331, capacity: 21, power: 450, controllable: false },
	],
};

export const BatteryCharging = Template.bind({});
BatteryCharging.args = {
	gridConfigured: true,
	pvConfigured: true,
	batteryConfigured: true,
	pvPower: 5000,
	gridPower: -1300,
	homePower: 800,
	loadpointsCompact: [
		{
			power: 1400,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 1400,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
	batteryPower: -1500,
	batterySoc: 75,
};

export const GridPVAndBattery = Template.bind({});
GridPVAndBattery.args = {
	gridConfigured: true,
	pvConfigured: true,
	batteryConfigured: true,
	pvPower: 1000,
	gridPower: 700,
	homePower: 3300,
	batteryPower: 1500,
	batterySoc: 30,
};

export const BatteryThresholds = Template.bind({});
BatteryThresholds.args = {
	gridConfigured: true,
	pvConfigured: true,
	batteryConfigured: true,
	pvPower: 8700,
	gridPower: -500,
	loadpointsCompact: [
		{
			power: 5000,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 5000,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 2500,
			icon: "bus",
			charging: true,
			title: "Garage",
			chargePower: 2500,
			connected: true,
			index: 1,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
	batteryPower: -700,
	batterySoc: 95,
};

export const PVThresholds = Template.bind({});
PVThresholds.args = {
	gridConfigured: true,
	pvConfigured: true,
	batteryConfigured: true,
	pvPower: 300,
	gridPower: 6500,
	homePower: 1000,
	loadpointsCompact: [
		{
			power: 5000,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 5000,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 1600,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 1600,
			connected: true,
			index: 1,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
	batteryPower: 800,
	batterySoc: 76,
};

export const GridOnly = Template.bind({});
GridOnly.args = {
	gridConfigured: true,
	pvConfigured: true,
	batteryConfigured: true,
	pvPower: 0,
	gridPower: 6500,
	homePower: 1000,
	loadpointsCompact: [
		{
			power: 5500,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 5500,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 0,
			icon: "car",
			charging: false,
			title: "Garage",
			chargePower: 0,
			connected: false,
			index: 1,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 0,
			icon: "car",
			charging: false,
			title: "Garage",
			chargePower: 0,
			connected: false,
			index: 2,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 0,
			icon: "car",
			charging: false,
			title: "Garage",
			chargePower: 0,
			connected: false,
			index: 3,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
	batteryPower: 0,
	batterySoc: 0,
};

export const LowPower = Template.bind({});
LowPower.args = {
	gridConfigured: true,
	pvConfigured: true,
	batteryConfigured: true,
	pvPower: 700,
	gridPower: -300,
	homePower: 300,
	batteryPower: -100,
	batterySoc: 55,
	tariffGrid: 0.25,
	tariffFeedIn: 0.08,
	currency: CURRENCY.EUR,
};

export const CO2 = Template.bind({});
CO2.args = {
	gridConfigured: true,
	pvConfigured: true,
	pvPower: 7300,
	gridPower: -2300,
	homePower: 800,
	loadpointsCompact: [
		{
			power: 1000,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 1000,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 1000,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 1000,
			connected: true,
			index: 1,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
		{
			power: 2200,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 2200,
			connected: true,
			index: 2,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
	tariffGrid: 0.25,
	tariffFeedIn: 0.08,
	tariffCo2: 723,
	smartCostType: "co2",
	currency: CURRENCY.EUR,
	pv: [{ power: 5000 }, { power: 2300 }],
};

export const UnknownInput = Template.bind({});
UnknownInput.args = {
	gridConfigured: true,
	pvConfigured: true,
	pvPower: 2000,
	gridPower: -2000,
	loadpointsCompact: [
		{
			power: 1000,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 1000,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
};

export const UnknownInputFill = Template.bind({});
UnknownInputFill.args = {
	gridConfigured: true,
	pvConfigured: true,
	batteryConfigured: true,
	pvPower: 500,
	gridPower: 0,
	batteryPower: -1000,
	loadpointsCompact: [],
};

export const UnknownOutput = Template.bind({});
UnknownOutput.args = {
	gridConfigured: true,
	pvConfigured: true,
	pvPower: 3000,
	gridPower: -1000,
	loadpointsCompact: [
		{
			power: 1700,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 1700,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
};

export const UnknownOutputLessThan10Percent = Template.bind({});
UnknownOutputLessThan10Percent.args = {
	gridConfigured: true,
	pvConfigured: true,
	pvPower: 3000,
	gridPower: -1000,
	loadpointsCompact: [
		{
			power: 1800,
			icon: "car",
			charging: true,
			title: "Garage",
			chargePower: 1800,
			connected: true,
			index: 0,
			vehicleName: "",
			vehicleSoc: 50,
			chargerFeatureHeating: false,
		},
	],
};
