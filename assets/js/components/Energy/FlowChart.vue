<template>
	<div ref="chartEl" class="flow-chart" data-testid="flow-chart"></div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { FONT_FAMILY, tooltipStyle, tooltipTable } from "./echarts";
import echartsChart from "@/mixins/echartsChart";
import formatter from "@/mixins/formatter";
import colors, { setAlpha } from "@/colors";
import { groupColor } from "./groups";
import type { Flow, FlowSink, FlowSource } from "./types";

const SOURCES: FlowSource[] = ["pv", "battery", "grid"];
const SINKS: FlowSink[] = ["home", "loadpoint", "battery", "export"];
// one shade per source, feed-in stands out
const LINK_ALPHA: Record<string, string> = { pv: "b0", battery: "50", grid: "40", export: "e0" };

// links below this share of all energy are noise, e.g. a trace of battery energy
// in the export from meter timing
const MIN_LINK_SHARE = 0.01;

const COMPACT = window.matchMedia("(max-width: 575.98px)");

interface Node {
	name: string;
	label: string;
	value: number;
	color: string | object;
	source: boolean;
}

export default defineComponent({
	name: "FlowChart",
	mixins: [formatter, echartsChart],
	props: {
		flows: { type: Array as PropType<Flow[]>, default: () => [] },
		loadpointLabels: { type: Array as PropType<string[]>, default: () => [] },
	},
	data() {
		return { compact: COMPACT.matches };
	},
	computed: {
		// without the hairline links, a node left with none is not drawn either
		visibleFlows(): Flow[] {
			const total = this.flows.reduce((acc, f) => acc + f.energy, 0);
			return this.flows.filter((f) => f.energy >= total * MIN_LINK_SHARE);
		},
		nodeColors(): Record<string, string> {
			return {
				pv: groupColor("pv"),
				battery: groupColor("battery"),
				grid: colors.grid || "",
				export: colors.export || "",
			};
		},
		nodes(): Node[] {
			// only what is drawn counts, so a node's label is the sum of the links you can hover
			const sum = (key: "from" | "to", id: string) =>
				this.visibleFlows
					.filter((f) => f[key] === id)
					.reduce((acc, f) => acc + f.energy, 0);
			const src = SOURCES.map((id) => ({
				name: `from-${id}`,
				label: this.$t(`energy.flow.${id}`),
				value: sum("from", id),
				color: this.nodeColors[id] || "",
				source: true,
			}));
			const loadpointLabel = this.loadpointLabels.join(" &\n");
			const dst = SINKS.map((id) => ({
				name: `to-${id}`,
				label: id === "loadpoint" ? loadpointLabel : this.$t(`energy.flow.${id}`),
				value: sum("to", id),
				// colored by where the energy came from, only feed-in keeps its own color
				color: id === "export" ? this.nodeColors[id] || "" : this.sourceBands(id),
				source: false,
			}));
			return [...src, ...dst].filter((n) => n.value > 0);
		},
		chartOption(): Record<string, unknown> {
			const compact = this.compact;
			const labelStyle = {
				fontFamily: FONT_FAMILY,
				rich: {
					name: {
						fontWeight: "bold",
						fontSize: compact ? 12 : 14,
						lineHeight: compact ? 15 : 17,
						color: colors.text || "",
					},
					value: {
						fontSize: compact ? 11 : 13,
						lineHeight: compact ? 15 : 17,
						color: colors.muted || "",
					},
				},
			};
			const labelFor = (n: Node) => ({
				...labelStyle,
				position: n.source ? "left" : "right",
				align: n.source ? "right" : "left",
				formatter: [
					...n.label.split("\n").map((l) => `{name|${l}}`),
					`{value|${this.fmtKWh(n.value)}}`,
				].join("\n"),
			});
			const labels = Object.fromEntries(this.nodes.map((n) => [n.name, n.label]));
			const totals = Object.fromEntries(this.nodes.map((n) => [n.name, n.value]));

			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				tooltip: {
					trigger: "item",
					...tooltipStyle(colors.text || ""),
					formatter: (params: {
						dataType: string;
						name: string;
						value: number;
						data: { source?: string; target?: string };
					}) => {
						const head =
							params.dataType === "edge"
								? `${labels[params.data.source || ""]} → ${labels[params.data.target || ""]}`
								: labels[params.name];
						const value =
							params.dataType === "edge" ? params.value : totals[params.name];
						return tooltipTable(head, [{ values: [this.fmtKWh(value)] }]);
					},
				},
				series: [
					{
						type: "sankey",
						layoutIterations: 0,
						nodeAlign: "justify",
						nodeWidth: 12,
						// at least the two-line label height so labels of tiny neighbours never overlap
						nodeGap: compact ? 32 : 36,
						draggable: false,
						left: compact ? 84 : 120,
						right: compact ? 96 : 120,
						top: 16,
						bottom: 20,
						// link colors carry their own alpha
						lineStyle: { curveness: 0.5, opacity: 1 },
						emphasis: { focus: "adjacency" },
						data: this.nodes.map((n) => ({
							name: n.name,
							value: n.value,
							depth: n.source ? 0 : 1,
							// rounded towards the label, sharp where the links attach
							itemStyle: {
								color: n.color,
								borderRadius: n.source ? [4, 0, 0, 4] : [0, 4, 4, 0],
							},
							label: labelFor(n),
						})),
						links: this.visibleFlows.map((f) => ({
							source: `from-${f.from}`,
							target: `to-${f.to}`,
							value: f.energy,
							lineStyle: { color: this.linkColor(f) },
						})),
					},
				],
			};
		},
	},
	mounted() {
		COMPACT.addEventListener("change", this.onMediaChange);
	},
	beforeUnmount() {
		COMPACT.removeEventListener("change", this.onMediaChange);
	},
	methods: {
		onMediaChange(e: MediaQueryListEvent) {
			this.compact = e.matches;
		},
		// the sink's sources as hard-edged bands from top to bottom, matching the
		// incoming links which attach in source order
		sourceBands(to: FlowSink): object {
			const flows = SOURCES.flatMap((from) =>
				this.visibleFlows.filter((f) => f.from === from && f.to === to)
			);
			const total = flows.reduce((acc, f) => acc + f.energy, 0) || 1;
			let offset = 0;
			const colorStops = flows.flatMap((f) => {
				const color = this.nodeColors[f.from] || "";
				const start = offset;
				offset = Math.min(1, offset + f.energy / total);
				return [
					{ offset: start, color },
					{ offset, color },
				];
			});
			return { type: "linear", x: 0, y: 0, x2: 0, y2: 1, colorStops };
		},
		linkColor(f: Flow): string {
			const key = f.to === "export" ? "export" : f.from;
			return setAlpha(this.nodeColors[key] || "", LINK_ALPHA[key] || "80") ?? "";
		},
	},
});
</script>

<style scoped>
/* fills the parent, which sizes it like the bar chart */
.flow-chart {
	height: 100%;
}
</style>
