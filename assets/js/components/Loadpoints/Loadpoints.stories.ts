import Loadpoints from "./Loadpoints.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

function loadpoint(opts = {}) {
	const base = {
		id: 0,
		pvConfigured: true,
		chargePower: 2800,
		chargedEnergy: 11e3,
		chargeDuration: 95 * 60,
		vehicleName: "tesla",
		vehicleTitle: "Tesla Model 3",
		enabled: true,
		connected: true,
		mode: "pv",
		charging: true,
		vehicleSoc: 66,
		limitSoc: 90,
		offeredCurrent: 7,
		minCurrent: 6,
		maxCurrent: 16,
		activePhases: 2,
		icon: "car",
		title: "Garage",
		power: 2100,
		index: 0,
		chargerFeatureHeating: false,
	};
	return { ...base, ...opts };
}

export default {
	title: "Loadpoints/Loadpoints",
	component: Loadpoints,
	parameters: {
		layout: "centered",
	},
} as Meta<typeof Loadpoints>;

const Template: StoryFn<typeof Loadpoints> = (args) => ({
	components: { Loadpoints },
	setup() {
		return { args };
	},
	template: '<Loadpoints v-bind="args" />',
});

export const Standard = Template.bind({});
Standard.args = {
	vehicles: [],
	loadpoints: [
		loadpoint({ title: "Carport", index: 0 }),
		loadpoint({
			title: "Water Heater",
			chargerFeatureIntegratedDevice: true,
			chargerIcon: "waterheater",
			index: 1,
		}),
	],
};
