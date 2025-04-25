import Loadpoint from "./Loadpoint.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
	title: "Loadpoints/Loadpoint",
	component: Loadpoint,
	parameters: {
		layout: "centered",
	},
} as Meta<typeof Loadpoint>;

const baseState = {
	id: 0,
	pvConfigured: true,
	chargePower: 2800,
	chargedEnergy: 11e3,
	chargeDuration: 95 * 60,
	vehicleTitle: "Mein Auto",
	vehicleName: "meinauto",
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
};

const Template: StoryFn<typeof Loadpoint> = (args) => {
	const story = () => ({
		components: { Loadpoint },
		setup() {
			return { args };
		},
		template: '<Loadpoint v-bind="args" />',
	});
	story.args = { ...baseState, ...args };
	return story;
};

export const Standard = Template.bind({});

export const WithoutSoc = Template.bind({});
WithoutSoc.args = {
	vehicleName: "",
};

export const Idle = Template.bind({});
Idle.args = {
	enabled: false,
	connected: false,
	vehicleName: "",
	mode: "off",
	charging: false,
	offeredCurrent: 0,
};

export const DisabledLongTitle = Template.bind({});
DisabledLongTitle.args = {
	title: "Charging point with a very very very long title!!!1!",
	remoteDisabled: "soft",
	remoteDisabledSource: "Sunny Home Manager",
	mode: "now",
	enabled: false,
	charging: false,
	chargePower: 0,
};

export const ChargerIconNoVehicle = Template.bind({});
ChargerIconNoVehicle.args = {
	chargerIcon: "heater",
	title: "Heating device with long name",
	mode: "now",
	chargerFeatureIntegratedDevice: true,
};
