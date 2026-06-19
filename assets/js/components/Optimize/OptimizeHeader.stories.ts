import OptimizeHeader from "./OptimizeHeader.vue";
import { CURRENCY, OptimizationStatus } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Optimize/OptimizeHeader",
  component: OptimizeHeader,
  parameters: {
    layout: "padded",
  },
} as Meta<typeof OptimizeHeader>;

const base = {
  updated: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
  horizonHours: 47,
  currency: CURRENCY.EUR,
  chargingStrategies: ["charge_before_export", "attenuate_grid_peaks", "none"],
  selectedStrategy: "charge_before_export",
  pending: false,
};

const Template: StoryFn<typeof OptimizeHeader> = (args) => ({
  components: { OptimizeHeader },
  setup() {
    return { args };
  },
  template: '<div class="container px-0"><OptimizeHeader v-bind="args" /></div>',
});

// you pay the grid (normal case): neutral color
export const Payment = Template.bind({});
Payment.args = { ...base, status: OptimizationStatus.OPTIMAL, netCost: 12.4 };

// solar surplus, you receive a credit: green, signed
export const Credit = Template.bind({});
Credit.args = { ...base, status: OptimizationStatus.OPTIMAL, netCost: -3.8 };

// solver could not produce a plan
export const Infeasible = Template.bind({});
Infeasible.args = { ...base, status: OptimizationStatus.INFEASIBLE, netCost: 0 };
