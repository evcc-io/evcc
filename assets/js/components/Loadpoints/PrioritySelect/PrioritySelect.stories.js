import PrioritySelect from "./PrioritySelect.vue";

export default {
  title: "Loadpoints/PrioritySelect",
  component: PrioritySelect,
};

const Template = (args) => ({
  components: { PrioritySelect },
  setup() {
    return { args };
  },
  template: '<PrioritySelect v-bind="args" />',
});

// Single icon story that can be controlled via args
export const SinglePrioritySelect = Template.bind({});
PrioritySelect.args = {
  priority: 0,
  editable: true,
};

export const AllPrios = (args) => ({
  components: { PrioritySelect },
  setup() {
    const prios = [-5, -3, -2, -1, 0, 1, 2, 3, 5];
    return { prios, args };
  },
  template: `
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 30px;">
      <div v-for="prio in prios" :key="prio" style="display: flex; flex-direction: column; align-items: center; gap: 10px;">
        <PrioritySelect :priority="prio" />
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
