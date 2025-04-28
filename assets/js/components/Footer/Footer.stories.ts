import Footer from "./Footer.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
	title: "Footer/Footer",
	component: Footer,
	argTypes: {
		version: { control: "object" },
		savings: { control: "object" },
	},
} as Meta<typeof Footer>;

const Template: StoryFn<typeof Footer> = (args) => ({
	components: { Footer },
	setup() {
		return { args };
	},
	template: '<Footer v-bind="args" />',
});

export const Default = Template.bind({});
Default.args = {
	version: {
		installed: "0.401",
		available: "0.401",
	},
	savings: {
		statistics: {
			"30d": { solarPercentage: 50 },
		},
	},
};

export const UpdateNightly = Template.bind({});
UpdateNightly.args = {
	version: {
		installed: "0.400",
		available: "0.400",
		commit: "5ce7be4",
	},
	savings: {
		statistics: {
			"30d": { solarPercentage: 22 },
		},
	},
};

export const UpdateAvailable = Template.bind({});
UpdateAvailable.args = {
	version: {
		installed: "0.400",
		available: "0.500",
	},
	savings: {
		statistics: {
			"30d": { solarPercentage: 100 },
		},
	},
};
