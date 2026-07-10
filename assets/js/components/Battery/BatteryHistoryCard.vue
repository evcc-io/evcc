<template>
	<Card edge-to-edge>
		<template #nav>
			<WindowNav
				:label="windowLabel"
				:prev-disabled="prevDisabled"
				:next-disabled="nextDisabled"
				@prev="changeOffset(-1)"
				@next="changeOffset(1)"
			/>
		</template>
		<template #actions>
			<SelectGroup
				v-if="showUnitToggle"
				id="batteryUnit"
				:options="unitOptions"
				:modelValue="effectiveUnit"
				@update:model-value="updateUnit"
			/>
		</template>
		<BatteryHistoryChart
			:batteries="batteries"
			:mode="effectiveUnit"
			:win-start="winStart.getTime()"
			:win-end="winEnd.getTime()"
			:now="now.getTime()"
			:has-forecast="windowHasForecast"
			:day-offset="dayOffset"
			:focused="focusedBattery"
		/>
		<LegendList
			v-if="batteries.length > 1"
			class="mt-4 mb-0"
			:legends="chartLegends"
			@focus="onLegendFocus"
		/>
	</Card>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import settings from "@/settings";
import { batteryColor } from "@/colors";
import type { Legend } from "../Sessions/types";
import type { BatterySeries } from "./types";
import Card from "../Helper/Card.vue";
import SelectGroup from "../Helper/SelectGroup.vue";
import LegendList from "../Sessions/LegendList.vue";
import WindowNav from "./WindowNav.vue";
import BatteryHistoryChart from "./BatteryHistoryChart.vue";

const HOUR = 3600 * 1000;
const MIN_OFFSET = -30; // page back ~30 days

// History + forecast chart with its date navigation, unit toggle and legend. Presentation
// only: paging, focus and the %/kWh unit are local view state; battery data is fed in via
// props (fetching stays in the page).
export default defineComponent({
	name: "BatteryHistoryCard",
	components: { Card, SelectGroup, LegendList, WindowNav, BatteryHistoryChart },
	mixins: [formatter],
	props: {
		batteries: { type: Array as PropType<BatterySeries[]>, default: () => [] },
		now: { type: Object as PropType<Date>, required: true },
		kwhAvailable: Boolean,
	},
	emits: ["range-start"],
	data() {
		return {
			dayOffset: 0,
			focusedBattery: null as number | null,
			selectedUnit: (settings.batteryUnit || "soc") as "soc" | "energy",
		};
	},
	computed: {
		// kWh (stacked energy) only makes sense across several batteries, so the toggle
		// appears only with more than one battery that all report a capacity
		showUnitToggle(): boolean {
			return this.batteries.length > 1 && this.kwhAvailable;
		},
		effectiveUnit(): "soc" | "energy" {
			return this.selectedUnit === "energy" && this.showUnitToggle ? "energy" : "soc";
		},
		unitOptions() {
			return [
				{ value: "soc", name: "%" },
				{ value: "energy", name: "kWh" },
			];
		},
		hasForecastData(): boolean {
			return this.batteries.some((b) => b.forecast.length > 0);
		},
		windowHasForecast(): boolean {
			return this.hasForecastData && this.winEnd > this.now;
		},
		winStart(): Date {
			const baseStartH = this.hasForecastData ? 24 : 48;
			return new Date(this.now.getTime() - baseStartH * HOUR + this.dayOffset * 24 * HOUR);
		},
		winEnd(): Date {
			const baseEndH = this.hasForecastData ? 24 : 0;
			return new Date(this.now.getTime() + baseEndH * HOUR + this.dayOffset * 24 * HOUR);
		},
		windowLabel(): string {
			// short weekday + day, no month (e.g. "Sa. 27. – Mo. 29.")
			const fmt = (d: Date) => `${this.weekdayShort(d)} ${d.getDate()}.`;
			return `${fmt(this.winStart)} – ${fmt(this.winEnd)}`;
		},
		prevDisabled(): boolean {
			return this.dayOffset <= MIN_OFFSET;
		},
		// one slide forward reveals the second half of the 48h optimizer forecast
		maxOffset(): number {
			return this.hasForecastData ? 1 : 0;
		},
		nextDisabled(): boolean {
			return this.dayOffset >= this.maxOffset;
		},
		chartLegends(): Legend[] {
			const focused = this.focusedBattery;
			return this.batteries.map((b, i) => ({
				label: b.title,
				color: batteryColor(i),
				type: "area",
				focusable: true,
				focusKey: i,
				dim: focused !== null && focused !== i,
				value:
					this.effectiveUnit === "energy"
						? this.fmtWh((b.currentSoc / 100) * b.capacity * 1e3, POWER_UNIT.AUTO)
						: this.fmtPercentage(b.currentSoc),
			}));
		},
	},
	watch: {
		// report the visible start so the page can keep enough history loaded (initial + paging)
		winStart: {
			handler(start: Date) {
				this.$emit("range-start", start);
			},
			immediate: true,
		},
		// drop a focus that points past the end after a battery is removed (else chart blanks)
		"batteries.length"(len: number) {
			if (this.focusedBattery !== null && this.focusedBattery >= len) {
				this.focusedBattery = null;
			}
		},
	},
	methods: {
		updateUnit(value: string | number | boolean | null) {
			this.selectedUnit = value === "energy" ? "energy" : "soc";
			settings.batteryUnit = this.selectedUnit;
		},
		changeOffset(dir: number) {
			this.dayOffset = Math.min(this.maxOffset, Math.max(MIN_OFFSET, this.dayOffset + dir));
		},
		onLegendFocus(legend: Legend) {
			const key = legend.focusKey as number;
			this.focusedBattery = this.focusedBattery === key ? null : key;
		},
	},
});
</script>
