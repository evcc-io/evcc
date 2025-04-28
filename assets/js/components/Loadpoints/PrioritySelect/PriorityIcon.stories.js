import PriorityIcon from "./PriorityIcon.vue";

export default {
  title: "Loadpoints/PriorityIcon",
  component: PriorityIcon,
  argTypes: {
    prio: {
      control: { type: "number", min: -10, max: 10 },
      description: "Number of icons to display",
    },
  },
};

const Template = (args) => ({
  components: { PriorityIcon },
  setup() {
    return { args };
  },
  template: '<PriorityIcon v-bind="args" />',
});

// Single icon story that can be controlled via args
export const SinglePriorityIcon = Template.bind({});
PriorityIcon.args = {
  prio: 0,
};

export const AllPrios = () => ({
  components: { PriorityIcon },
  setup() {
    const prios = [-5, -3, -2, -1, 0, 1, 2, 3, 5];
    return { prios };
  },
  template: `
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 30px;">
      <div v-for="prio in prios" :key="prio" style="display: flex; flex-direction: column; align-items: center; gap: 10px;">
        <PriorityIcon :prio="prio" />
        <small>{{ prio }}</small>
      </div>
    </div>
  `,
});
