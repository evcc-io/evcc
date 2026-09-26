import { computed, onUnmounted, reactive } from "vue";
import CircuitTags, { type CircuitLoadpoint } from "./CircuitTags.vue";
import DeviceCard from "./DeviceCard.vue";
import CircuitsIcon from "../MaterialIcon/Circuits.vue";
import { circuitTree } from "@/utils/circuits";
import type { Circuit } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Config/CircuitTags",
  component: CircuitTags,
} as Meta<typeof CircuitTags>;

const circuits: Record<string, Circuit> = {
  home: {
    name: "home",
    title: "Home",
    power: 8000,
    current: 12.2,
    maxPower: 20000,
    maxCurrent: 32,
  },
  carport: { name: "carport", title: "Carport", parent: "home", power: 1000, maxPower: 5000 },
  workshop: {
    name: "workshop",
    title: "Workshop in the basement",
    parent: "home",
    power: 18000,
    maxPower: 22000,
    current: 15,
    maxCurrent: 10,
  },
  garage: {
    name: "garage",
    title: "Garage",
    parent: "home",
    power: 4000,
    current: 15.4,
    maxPower: 11000,
    maxCurrent: 16,
  },
  corner: {
    name: "corner",
    title: "Corner",
    parent: "garage",
    power: 1400,
    current: 6,
    maxCurrent: 16,
  },
};

const loadpoints: CircuitLoadpoint[] = [
  { name: "lp1", title: "Carport", circuit: "carport", power: 1000 },
  { name: "lp2", title: "Heat pump", circuit: "workshop", power: 2500 },
  { name: "lp3", title: "Wallbox left", circuit: "garage", power: 2000 },
  { name: "lp4", title: "Wallbox right", circuit: "garage", power: 0 },
  { name: "lp5", title: "Motorbike", circuit: "corner", power: 1400 },
];

const card = `
  <div class="p-4" style="max-width: 400px">
    <DeviceCard title="Load Management" editable>
      <template #icon><CircuitsIcon /></template>
      <template #tags>
        <CircuitTags :nodes="[root]" :loadpoints="loadpoints" />
      </template>
    </DeviceCard>
  </div>
`;

const components = { CircuitTags, DeviceCard, CircuitsIcon };

export const Tree: StoryFn = () => ({
  components,
  setup() {
    return { root: circuitTree(circuits), loadpoints };
  },
  template: card,
});

// values change every second, columns must not shift
export const Live: StoryFn = () => ({
  components,
  setup() {
    const state = reactive(structuredClone(circuits));
    const lps = reactive(structuredClone(loadpoints));
    const timer = setInterval(() => {
      for (const c of Object.values(state)) {
        c.power = Math.random() * (c.maxPower ?? 20000) * 1.1;
        if (c.maxCurrent) c.current = Math.random() * c.maxCurrent * 1.1;
      }
      for (const lp of lps) lp.power = Math.random() * 11000;
    }, 1000);
    onUnmounted(() => clearInterval(timer));
    return { root: computed(() => circuitTree(state)), loadpoints: lps };
  },
  template: card,
});
