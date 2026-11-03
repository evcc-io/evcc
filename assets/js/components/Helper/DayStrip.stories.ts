import DayStrip, { type Day, type DayState } from "./DayStrip.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Helper/DayStrip",
  component: DayStrip,
} as Meta<typeof DayStrip>;

const month = (today: number, warning: number[] = []): Day[] =>
  Array.from({ length: 30 }, (_, i) => {
    const day = i + 1;
    const date = new Date(2026, 10, day);
    const state: DayState = day > today ? "none" : warning.includes(day) ? "warning" : "ok";
    return {
      date,
      state,
      value: state === "none" ? undefined : state === "warning" ? "max 23.4 kW" : "max 8.2 kW",
      to: "/",
    };
  });

const Template: StoryFn<typeof DayStrip> = (args) => ({
  components: { DayStrip },
  setup() {
    return { args };
  },
  template: '<div class="p-4" style="max-width: 420px"><DayStrip v-bind="args" /></div>',
});

export const Ok = Template.bind({});
Ok.args = { days: month(18) };

export const Exceeded = Template.bind({});
Exceeded.args = { days: month(18, [4, 15]) };

export const FirstDay = Template.bind({});
FirstDay.args = { days: month(1) };

export const FullMonth = Template.bind({});
FullMonth.args = { days: month(30, [4, 11, 15, 22]) };
