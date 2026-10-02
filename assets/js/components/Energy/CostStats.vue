<template>
	<StatCards :stats="stats">
		<template #autarky>
			<MixBar :segments="autarkySegments" />
		</template>
	</StatCards>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import StatCards from "./StatCards.vue";
import MixBar from "./MixBar.vue";
import formatter from "@/mixins/formatter";
import { CURRENCY } from "@/types/evcc";
import type { FlowCo2, FlowCost, StatItem } from "./types";

export default defineComponent({
	name: "CostStats",
	components: { StatCards, MixBar },
	mixins: [formatter],
	props: {
		autarky: { type: Number, default: 0 }, // percent
		cost: { type: Object as PropType<FlowCost> },
		co2: { type: Object as PropType<FlowCo2> },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
	},
	computed: {
		// the remainder is the track, same tone as the bootstrap progress track
		autarkySegments(): { value: number; color: string }[] {
			return [
				{ value: this.autarky, color: "var(--evcc-accent2)" },
				{ value: 100 - this.autarky, color: "var(--bs-secondary-bg)" },
			];
		},
		stats(): StatItem[] {
			const list: StatItem[] = [
				{
					key: "autarky",
					accent: "text-accent2",
					label: this.$t("energy.stat.autarky"),
					number: this.autarky,
					format: (v: number) => this.fmtPercentage(v),
				},
			];
			if (this.cost) {
				// consumption priced grid only minus what it cost with own energy at the feed-in price
				const savings = this.cost.baseline - this.cost.consumption;
				const consumption = this.cost.consumptionEnergy;
				const effective = consumption ? this.cost.consumption / consumption : 0;
				const fmt = (v: number) => this.fmtPricePerKWh(v, this.currency, false, false);
				list.push({
					key: "savings",
					accent: "text-accent2",
					label: this.$t("energy.stat.savings"),
					number: savings,
					format: (v: number) => this.fmtMoneyWithSymbol(v, this.currency),
					// effective price per kWh consumed vs. the average grid price
					sub: `${fmt(effective)} vs. ${this.fmtPricePerKWh(this.cost.avgGrid, this.currency)}`,
					tooltip: this.$t("energy.stat.savingsTooltip", {
						energy: this.fmtKWh(this.cost.consumptionEnergy),
					}),
				});
			} else {
				list.push({
					key: "savings",
					label: this.$t("energy.stat.savings"),
					empty: this.$t("energy.stat.noPrice"),
				});
			}
			if (this.co2) {
				const saved = this.co2.baseline - this.co2.consumption;
				const consumption = this.co2.consumptionEnergy;
				const effective = consumption ? (this.co2.consumption * 1000) / consumption : 0;
				list.push({
					key: "co2",
					accent: "text-accent3",
					label: this.$t("energy.stat.co2Saved"),
					number: saved * 1000,
					format: (v: number) => this.fmtGrams(v),
					sub: `${this.fmtNumber(effective, 0)} vs. ${this.fmtCo2Medium(this.co2.avgCo2)}`,
					tooltip: this.$t("energy.stat.co2SavedTooltip", {
						energy: this.fmtKWh(this.co2.consumptionEnergy),
					}),
				});
			} else {
				list.push({
					key: "co2",
					label: this.$t("energy.stat.co2Saved"),
					empty: this.$t("energy.stat.noCo2"),
				});
			}
			return list;
		},
	},
});
</script>
