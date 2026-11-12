<template>
	<StatRow :stats="stats" inline :selected="selected" @select="select" />
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import StatRow from "./StatRow.vue";
import formatter from "@/mixins/formatter";
import colors from "@/colors";
import { CURRENCY } from "@/types/evcc";
import type { FlowCost, StatItem } from "./types";

// what came from and went to the grid, with cost and revenue when there is a tariff
export default defineComponent({
	name: "GridStats",
	components: { StatRow },
	mixins: [formatter],
	props: {
		importEnergy: { type: Number, default: 0 }, // kWh, metered
		exportEnergy: { type: Number, default: 0 },
		cost: { type: Object as PropType<FlowCost> },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		// the direction the chart shows, set while it shows one of them only
		direction: String as PropType<"import" | "export">,
	},
	emits: ["update:direction"],
	computed: {
		selected(): string | undefined {
			if (!this.direction) return undefined;
			return this.direction === "export" ? "gridExport" : "gridImport";
		},
		stats(): StatItem[] {
			const cost = this.cost;
			const stat = (key: "energy" | "returnEnergy", energy: number, money?: number) => ({
				key: key === "energy" ? "gridImport" : "gridExport",
				label: this.$t(`energy.direction.grid.${key}`),
				number: energy,
				format: (v: number) => this.fmtKWh(v),
				sub: money === undefined ? "" : this.fmtMoneyWithSymbol(money, this.currency),
			});
			return [
				stat("energy", this.importEnergy, cost?.import),
				{
					...stat("returnEnergy", this.exportEnergy, cost?.export),
					// the yellow reads on dark only, on light the number keeps the text color
					accent: colors.dark ? "text-export" : undefined,
				},
			];
		},
	},
	methods: {
		select(key: string) {
			this.$emit("update:direction", key === "gridExport" ? "export" : "import");
		},
	},
});
</script>
