<template>
	<StatCards :stats="stats" />
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import StatCards from "./StatCards.vue";
import formatter from "@/mixins/formatter";
import { CURRENCY } from "@/types/evcc";
import type { FlowCost, StatItem } from "./types";

// what import cost and export earned, pointing at the tariff settings without one
export default defineComponent({
	name: "GridStats",
	components: { StatCards },
	mixins: [formatter],
	props: {
		cost: { type: Object as PropType<FlowCost> },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
	},
	computed: {
		stats(): StatItem[] {
			const cost = this.cost;
			if (!cost) {
				const empty = this.$t("energy.stat.noPrice");
				return [
					{ key: "gridImport", label: this.$t("energy.grid.cost"), empty },
					{ key: "gridExport", label: this.$t("energy.grid.revenue"), empty },
				];
			}
			const tile = (key: string, label: string, money: number, energy: number): StatItem => ({
				key,
				label,
				number: money,
				format: (v: number) => this.fmtMoneyWithSymbol(v, this.currency),
				sub: energy ? `ø ${this.fmtPricePerKWh(money / energy, this.currency)}` : "",
			});
			return [
				tile("gridImport", this.$t("energy.grid.cost"), cost.import, cost.importEnergy),
				{
					...tile(
						"gridExport",
						this.$t("energy.grid.revenue"),
						cost.export,
						cost.exportEnergy
					),
					accent: "text-accent1",
				},
			];
		},
	},
});
</script>
