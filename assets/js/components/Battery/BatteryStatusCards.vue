<template>
	<div v-if="showSplit" class="cards cards--split gap-3">
		<BatteryStatusCard v-bind="cards[0]" />
		<Card :title="$t('battery.optimizer.title')" data-testid="battery-optimizer-card">
			<OptimizerInfo :suggestion="suggestion" :forecast="batteryForecast" />
		</Card>
	</div>
	<div v-else class="cards gap-3">
		<BatteryStatusCard v-for="card in cards" :key="card.id" v-bind="card" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { batteryColor } from "@/colors";
import type { Battery, BatteryForecast, BatteryMeter } from "@/types/evcc";
import Card from "../Helper/Card.vue";
import BatteryStatusCard from "./BatteryStatusCard.vue";
import OptimizerInfo from "./OptimizerInfo.vue";
import type { BatterySuggestion, BatteryStatusCardModel } from "./types";

// One battery + forecast splits into battery and optimizer cards; several batteries prepend a
// combined aggregate. Stateless, derived from the battery object.
export default defineComponent({
	name: "BatteryStatusCards",
	components: { Card, BatteryStatusCard, OptimizerInfo },
	props: {
		battery: { type: Object as PropType<Battery> },
		suggestion: { type: Object as PropType<BatterySuggestion | null>, default: null },
	},
	computed: {
		devices(): BatteryMeter[] {
			return this.battery?.devices ?? [];
		},
		batteryForecast(): BatteryForecast | null {
			const fc = this.battery?.forecast;
			return fc?.highest || fc?.lowest ? fc : null;
		},
		showSplit(): boolean {
			return this.devices.length === 1 && !!this.batteryForecast;
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
				suggestion: null, // per-battery suggestion not wired yet
				forecast: null, // aggregate forecast lives on the combined / dedicated card
			}));
			// combined uses the site aggregate soc/power, not per-device sums
			if (multiple) {
				list.unshift({
					id: "combined",
					title: this.$t("battery.combined"),
					soc: this.battery?.soc ?? 0,
					power: this.battery?.power ?? 0,
					capacity: this.devices.reduce((s, d) => s + (d.capacity || 0), 0),
					color: batteryColor(0),
					suggestion: this.suggestion,
					forecast: this.batteryForecast,
				});
			}
			return list;
		},
	},
});
</script>

<style scoped>
.cards {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
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
