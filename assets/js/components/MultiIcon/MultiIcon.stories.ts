import { ICON_SIZE } from "@/types/evcc";
import MultiIcon from "./MultiIcon.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
	title: "MultiIcon/MultiIcon",
	component: MultiIcon,
	argTypes: {
		count: {
			control: { type: "number", min: 1, max: 10 },
			description: "Number of icons to display",
		},
		size: {
			options: Object.values(ICON_SIZE),
			control: { type: "select" },
			description: "Size of the icons",
		},
	},
} as Meta<typeof MultiIcon>;

export const SingleIcon: StoryFn<typeof MultiIcon> = (args) => ({
	components: { MultiIcon },
	setup() {
		return { args };
	},
	template: '<MultiIcon v-bind="args" />',
});

export const AllCounts: StoryFn<typeof MultiIcon> = (args) => ({
	components: { MultiIcon },
	setup() {
		const counts = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
		return { counts, args };
	},
	template: `
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 30px;">
      <div v-for="count in counts" :key="count" style="display: flex; flex-direction: column; align-items: center; gap: 10px;">
        <MultiIcon :count="count" :size="args.size" />
        <small>{{ count }}</small>
      </div>
    </div>
  `,
});

AllCounts.args = {
	size: ICON_SIZE.XL,
};
