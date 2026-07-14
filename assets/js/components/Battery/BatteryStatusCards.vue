<template>
	<div v-if="showSplit" class="cards cards--split gap-4">
		<BatteryStatusCard v-bind="cards[0]" :battery-mode="batteryMode" />
		<Card data-testid="battery-optimizer-card">
			<OptimizerInfo :suggestion="singleSuggestion" :forecast="batteryForecast" />
		</Card>
	</div>
	<div v-else class="cards gap-4">
		<BatteryStatusCard
			v-for="card in cards"
			:key="card.id"
			v-bind="card"
			:battery-mode="batteryMode"
		/>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { batteryColor } from "@/colors";
import type {
	BATTERY_MODE,
	Battery,
	BatteryForecast,
	BatteryMeter,
	BatterySuggestion,
} from "@/types/evcc";
import Card from "../Helper/Card.vue";
import BatteryStatusCard from "./BatteryStatusCard.vue";
import OptimizerInfo from "./OptimizerInfo.vue";
import type { BatteryStatusCardModel } from "./types";

// One battery + forecast splits into battery and optimizer cards; several batteries prepend a
// combined aggregate. Stateless, derived from the battery object.
export default defineComponent({
	name: "BatteryStatusCards",
	components: { Card, BatteryStatusCard, OptimizerInfo },
	props: {
		battery: { type: Object as PropType<Battery> },
		batteryMode: String as PropType<BATTERY_MODE>,
	},
	computed: {
		devices(): BatteryMeter[] {
			return this.battery?.devices ?? [];
		},
		batteryForecast(): BatteryForecast | null {
			const fc = this.battery?.forecast;
			return fc?.highest || fc?.lowest ? fc : null;
		},
		singleSuggestion(): BatterySuggestion | null {
			return this.deviceSuggestion(this.devices[0]);
		},
		showSplit(): boolean {
			return this.devices.length === 1 && !!(this.batteryForecast || this.singleSuggestion);
		},
		cards(): BatteryStatusCardModel[] {
			const multiple = this.devices.length > 1;
			const list: BatteryStatusCardModel[] = this.devices.map((d, i) => ({
				id: d.name || `battery-${i}`, // name is unique; title may repeat or be empty
				title: d.title || d.name || `Battery ${i + 1}`,
				soc: d.soc,
				power: d.power,
				capacity: d.capacity || 0,
				color: batteryColor(i),
				controllable: d.controllable,
				// single battery shows the suggestion on the dedicated optimizer card
				suggestion: multiple ? this.deviceSuggestion(d) : null,
				forecast: null, // aggregate forecast lives on the combined / dedicated card
			}));
			// combined uses the site aggregate soc/power, not per-device sums; the site-wide
			// battery mode always applies here regardless of per-device controllability
			if (multiple) {
				list.unshift({
					id: "combined",
					title: this.$t("battery.combined"),
					soc: this.battery?.soc ?? 0,
					power: this.battery?.power ?? 0,
					capacity: this.devices.reduce((s, d) => s + (d.capacity || 0), 0),
					color: batteryColor(0),
					controllable: true,
					suggestion: null, // no aggregate; device cards show their own
					forecast: this.batteryForecast,
				});
			}
			return list;
		},
	},
	methods: {
		deviceSuggestion(d?: BatteryMeter): BatterySuggestion | null {
			// optimizer only emits suggestions for controllable batteries
			return d?.suggestion?.actionable ? d.suggestion : null;
		},
	},
});
</script>

<style scoped>
.cards {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(min(380px, 100%), 1fr));
}
.cards--split {
	grid-template-columns: 1fr 1fr;
}
@media (max-width: 767.98px) {
	.cards--split {
		grid-template-columns: 1fr;
	}
}
</style>
