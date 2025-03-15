import MultiIcon from "./MultiIcon.vue";

export default {
  title: "MultiIcon/MultiIcon",
  component: MultiIcon,
  argTypes: {
    count: {
      control: { type: "number", min: 1, max: 10 },
      description: "Number of icons to display",
    },
    size: {
      control: { type: "select", options: ["sm", "md", "lg", "xl"] },
      description: "Size of the icons",
    },
  },
};

export const SingleIcon = (args) => ({
  components: { MultiIcon },
  setup() {
    return { args };
  },
  template: '<MultiIcon v-bind="args" />',
});

export const AllCounts = (args) => ({
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
  size: "xl",
};
