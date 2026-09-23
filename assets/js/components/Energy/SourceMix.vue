<template>
	<MixBar :segments="segments" :tooltip="rows" data-testid="source-mix" />
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import MixBar from "./MixBar.vue";
import type { TooltipRow } from "../Forecast/echarts";
import formatter from "@/mixins/formatter";
import colors, { darken, lighten } from "@/colors";
import type { Flow, FlowSink, FlowSource } from "./types";
import { SOURCES, sourceShares } from "./mix";

// pv, battery and grid share of `energy`, using the sink's ratio over the period.
// Lighter and darker shades of the device color, grid keeps its own
export default defineComponent({
	name: "SourceMix",
	components: { MixBar },
	mixins: [formatter],
	props: {
		flows: { type: Array as PropType<Flow[]>, default: () => [] },
		sink: { type: String as PropType<FlowSink>, required: true },
		energy: { type: Number, default: 0 }, // kWh
		color: { type: String, default: "" },
	},
	computed: {
		sources(): { from: FlowSource; share: number; color: string }[] {
			const shares = sourceShares(this.flows, this.sink);
			const color: Record<FlowSource, string> = {
				pv: lighten(this.color, 0.35),
				battery: darken(this.color, 0.8),
				grid: colors.grid || "",
			};
			return SOURCES.map((from) => ({ from, share: shares[from], color: color[from] }));
		},
		segments(): { value: number; color: string }[] {
			return this.sources.map((src) => ({ value: src.share, color: src.color }));
		},
		rows(): TooltipRow[] {
			return this.sources.map((src) => ({
				name: this.$t(`energy.consumers.source.${src.from}`),
				values: [
					this.fmtPercentage(src.share),
					this.fmtKWh((this.energy * src.share) / 100),
				],
			}));
		},
	},
});
</script>
