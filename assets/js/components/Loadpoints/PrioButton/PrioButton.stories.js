import PrioButton from "./PrioButton.vue";

export default {
  title: "Loadpoints/PrioButton",
  component: PrioButton,
  argTypes: {
    prio: {
      control: { type: "number", min: -10, max: 10 },
      description: "Number of icons to display",
    },
    size: {
      control: "select",
      options: ["sm", "md", "lg", "xl"],
      defaultValue: "xl",
    },
  },
};

const Template = (args) => ({
  components: { PrioButton },
  setup() {
    return { args };
  },
  template: '<PrioButton v-bind="args" />',
});

// Single icon story that can be controlled via args
export const SinglePrioButton = Template.bind({});
PrioButton.args = {
  priority: 0,
  size: "xl",
  editable: true,
};

export const AllPrios = (args) => ({
  components: { PrioButton },
  setup() {
    const prios = [-5, -3, -2, -1, 0, 1, 2, 3, 5];
    return { prios, args };
  },
  template: `
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 30px;">
      <div v-for="prio in prios" :key="prio" style="display: flex; flex-direction: column; align-items: center; gap: 10px;">
        <PrioButton :prio="prio" :size="args.size" :editable="args.editable" />
        <small>{{ prio }}</small>
      </div>
    </div>
  `,
});
/* 
AllCounts.args = {
  size: "xl",
};
 */
