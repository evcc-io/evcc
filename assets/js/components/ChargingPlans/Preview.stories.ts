import { CURRENCY, type Rate } from "assets/js/types/evcc";
import Preview from "./Preview.vue";
import type { StoryFn } from "@storybook/vue3";

const now = new Date();

function createDate(hoursFromNow: number) {
	const result = new Date(now.getTime());
	result.setHours(result.getHours() + hoursFromNow);
	return result;
}

function createRate(value: number, hoursFromNow: number, durationHours = 1): Rate {
	const start = new Date(now.getTime());
	start.setHours(start.getHours() + hoursFromNow);
	start.setMinutes(0);
	start.setSeconds(0);
	start.setMilliseconds(0);
	const end = new Date(start.getTime());
	end.setHours(start.getHours() + durationHours);
	end.setMinutes(0);
	end.setSeconds(0);
	end.setMilliseconds(0);
	return { start, end, value };
}

// Scenario data
const co2Data = {
	rates: [
		545, 518, 545, 518, 0, 545, 527, 527, 536, 518, 400, 336, 336, 339, 344, 336, 336, 336, 372,
		400, 555, 555, 545, 555, 564, 545, 555, 545, 536, 545, 527, 536, 518, 545, 509, 336, 336,
		336,
	].map((value, i) => createRate(value, i)),
	duration: 8695,
	plan: [createRate(213, 4), createRate(336, 11), createRate(336, 12)],
	smartCostType: "co2",
	targetTime: createDate(14),
};

const fixedData = {
	rates: [createRate(0.442, 0, 50)],
	duration: 8695,
	plan: [createRate(0.442, 12, 3)],
	smartCostType: "price",
	currency: CURRENCY.EUR,
	targetTime: createDate(14),
};

const zonedData = {
	rates: [
		createRate(3.72, 0, 4),
		createRate(2.39, 4, 12),
		createRate(3.72, 16, 12),
		createRate(2.39, 28, 12),
		createRate(3.72, 40, 12),
	],
	duration: 8695,
	plan: [createRate(2.39, 13, 3)],
	smartCostType: "price",
	currency: CURRENCY.DKK,
	targetTime: createDate(17),
};

const unknownData = {
	rates: co2Data.rates.slice(0, 16),
	duration: 8695,
	plan: [createRate(213, 4), createRate(336, 11), createRate(336, 12)],
	smartCostType: "co2",
	targetTime: createDate(14),
};

const dynamicData = {
	rates: [
		0.12, 0.15, 0, -0.05, -0.11, -0.24, -0.08, 0.12, 0.25, 0.29, 0.22, 0.31, 0.31, 0.33,
	].map((value, i) => createRate(value, i)),
	duration: 8695,
	plan: [createRate(0.23, 2, 5)],
	smartCostType: "price",
	currency: CURRENCY.EUR,
	targetTime: createDate(13),
};

export default {
	title: "ChargingPlans/Preview",
	component: Preview,
	argTypes: {
		rates: { control: "object" },
		duration: { control: "number" },
		plan: { control: "object" },
		smartCostType: { control: "select", options: ["co2", "price"] },
		currency: { control: "text" },
		targetTime: { control: "text" },
	},
};

const Template: StoryFn<typeof Preview> = (args) => ({
	components: { Preview },
	setup() {
		return { args };
	},
	template: '<Preview v-bind="args" />',
});

export const Co2 = Template.bind({});
Co2.args = co2Data;

export const Fixed = Template.bind({});
Fixed.args = fixedData;

export const Zoned = Template.bind({});
Zoned.args = zonedData;

export const Unknown = Template.bind({});
Unknown.args = unknownData;

export const Dynamic = Template.bind({});
Dynamic.args = dynamicData;
