import BatteryIcon from "./BatteryIcon.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
	title: "Energyflow/BatteryIcon",
	component: BatteryIcon,
	parameters: {
		layout: "centered",
	},
} as Meta<typeof BatteryIcon>;

const Template: StoryFn<typeof BatteryIcon> = (args) => {
	const story = () => ({
		components: { BatteryIcon },
		setup() {
			return { args };
		},
		template: '<BatteryIcon v-bind="args" />',
	});
	story.args = args;
	return story;
};

export const Empty = Template.bind({});
Empty.args = { soc: 0 };

export const Soc10 = Template.bind({});
Soc10.args = { soc: 10 };

export const Soc20 = Template.bind({});
Soc20.args = { soc: 20 };

export const Soc30 = Template.bind({});
Soc30.args = { soc: 30 };

export const Soc40 = Template.bind({});
Soc40.args = { soc: 40 };

export const Soc50 = Template.bind({});
Soc50.args = { soc: 50 };

export const Soc60 = Template.bind({});
Soc60.args = { soc: 60 };

export const Soc70 = Template.bind({});
Soc70.args = { soc: 70 };

export const Soc80 = Template.bind({});
Soc80.args = { soc: 80 };

export const Soc90 = Template.bind({});
Soc90.args = { soc: 90 };

export const Hold = Template.bind({});
Hold.args = { hold: true };

export const GridCharge = Template.bind({});
GridCharge.args = { gridCharge: true };
