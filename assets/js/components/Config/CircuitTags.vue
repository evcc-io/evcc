<template>
	<div class="circuit-tags" :class="{ 'circuit-tags--child': depth > 0 }">
		<div v-for="node in nodes" :key="node.name" class="node-block">
			<!-- Circuit header -->
			<div class="node-header">
				<span
					class="node-name"
					:class="{ 'node-name--root': depth === 0 }"
				>
					{{ node.title || node.name }}
				</span>
				<span v-if="depth > 0" class="node-power evcc-gray">
					{{ fmtW(node.power) }}
				</span>
			</div>

			<!-- Limits -->
			<div v-if="parts(node).length" class="metrics">
				<div
					v-for="part in parts(node)"
					:key="part.unit"
					class="metric"
				>
					<div class="metric-header">
						<span class="metric-name">
							{{ part.unit === "kW" ? "Leistung" : "Strom" }}
						</span>
						<span
							class="metric-value"
							:class="{ warning: part.warning }"
						>
							{{ part.label }}
						</span>
					</div>
					<div class="bar-track">
						<div
							class="bar-fill"
							:class="{ warning: part.warning }"
							:style="{ width: barWidth(part.ratio) + '%' }"
						/>
					</div>
				</div>
			</div>

			<!-- Root circuit without configured limits -->
			<div v-else-if="depth === 0" class="root-power evcc-gray">
				{{ fmtW(node.power) }}
			</div>

			<!-- Loadpoints assigned to this circuit -->
			<div v-if="loadpointsFor(node).length" class="loadpoints">
				<div
					v-for="lp in loadpointsFor(node)"
					:key="lp.name"
					class="loadpoint-row"
				>
					<span class="lp-name evcc-gray">
						<svg
							width="12"
							height="12"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linejoin="round"
						>
							<path d="M13 2 6 13h5l-1 9 7-11h-5l1-9z" />
						</svg>
						<span>
							{{ lp.title || lp.name }}
						</span>
					</span>
					<span class="lp-power evcc-gray">
						{{ fmtW(meterPower(lp)) }}
					</span>
				</div>
			</div>

			<!-- Child circuits -->
			<div v-if="node.children?.length" class="children">
				<CircuitTags
					:nodes="node.children"
					:loadpoints="loadpoints"
					:meters="meters"
					:depth="depth + 1"
				/>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import formatter from "@/mixins/formatter.ts";
import type { CircuitNode } from "@/utils/circuits.ts";
import type { ConfigLoadpoint, Meter } from "@/types/evcc";

interface LimitPart {
	unit: "kW" | "A";
	label: string;
	ratio: number;
	warning: boolean;
}

export default {
	name: "CircuitTags",
	mixins: [formatter],
	props: {
		nodes: {
			type: Array as PropType<CircuitNode[]>,
			required: true,
		},
		loadpoints: {
			type: Array as PropType<ConfigLoadpoint[]>,
			required: true,
		},
		meters: {
			type: Array as PropType<Meter[]>,
			required: true,
		},
		depth: {
			type: Number,
			default: 0,
		},
	},
	methods: {
		parts(node: CircuitNode): LimitPart[] {
			const result: LimitPart[] = [];

			if (node.maxPower !== undefined) {
				const power = node.power ?? 0;
				const ratio = node.maxPower > 0 ? power / node.maxPower : 0;

				result.push({
					unit: "kW",
					label: `${this.fmt1(power / 1000)} / ${this.fmt1(
						node.maxPower / 1000,
					)} kW`,
					ratio,
					warning: ratio >= 1,
				});
			}

			if (node.maxCurrent !== undefined) {
				const current = node.current ?? 0;
				const ratio =
					node.maxCurrent > 0 ? current / node.maxCurrent : 0;

				result.push({
					unit: "A",
					label: `${this.fmt1(current)} / ${this.fmt1(
						node.maxCurrent,
					)} A`,
					ratio,
					warning: ratio >= 1,
				});
			}

			return result;
		},
		fmt1(value: number): string {
			return (value ?? 0).toFixed(1);
		},
		barWidth(ratio: number): number {
			return Math.max(0, Math.min(100, ratio * 100));
		},
		loadpointsFor(node: CircuitNode): ConfigLoadpoint[] {
			return this.loadpoints.filter((lp) => lp.circuit === node.name);
		},
		meterPower(lp: ConfigLoadpoint): number {
			const meter = this.meters.find((m) => m.name === lp.meter);

			return meter?.power ?? 0;
		},
	},
};
</script>

<style scoped>
.circuit-tags {
	width: 100%;
	min-width: 0;
	box-sizing: border-box;
}
/*
 * Continuous hierarchy line per level.
 * The line is attached to the shared sibling container,
 * so it stays uninterrupted between child nodes.
 */
.circuit-tags--child {
	border-left: 2px solid var(--evcc-gray-10);
	padding-left: 12px;
}
.node-block {
	width: 100%;
	min-width: 0;
	box-sizing: border-box;
}
/*
 * Spacing between sibling circuits.
 * The hierarchy line remains continuous because
 * it belongs to the parent container.
 */
.node-block + .node-block {
	margin-top: 18px;
}
/*
 * Circuit header
 */
.node-header {
	display: flex;
	align-items: baseline;
	justify-content: space-between;
	gap: 12px;
	width: 100%;
	min-width: 0;
}
.node-name {
	min-width: 0;
	font-size: 14px;
	font-weight: 700;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.node-name--root {
	font-size: 15px;
}
.node-power {
	flex-shrink: 0;
	font-size: 13px;
	font-variant-numeric: tabular-nums;
	white-space: nowrap;
}
/*
 * Metrics
 */
.metrics {
	display: flex;
	flex-direction: column;
	gap: 9px;
	width: 100%;
	margin-top: 8px;
}
.metric {
	width: 100%;
	min-width: 0;
}
.metric-header {
	display: flex;
	align-items: baseline;
	justify-content: space-between;
	gap: 10px;
	width: 100%;
	min-width: 0;
	margin-bottom: 4px;
}
.metric-name {
	color: var(--evcc-gray);
	font-size: 10px;
	font-weight: 600;
}
.metric-value {
	flex-shrink: 0;
	font-size: 13px;
	font-variant-numeric: tabular-nums;
	text-align: right;
	white-space: nowrap;
}
/*
 * Progress bars
 */
.bar-track {
	width: 100%;
	height: 4px;
	overflow: hidden;
	border-radius: 2px;
	background: var(--evcc-gray-10);
}
.bar-fill {
	height: 100%;
	border-radius: inherit;
	background: var(--evcc-dark-green);
	transition: width 0.2s ease;
}
.warning {
	color: var(--bs-warning);
	font-weight: 700;
}
.bar-fill.warning {
	background: var(--bs-warning);
}
/*
 * Root circuit without limits
 */
.root-power {
	margin-top: 5px;
	font-size: 13px;
	font-variant-numeric: tabular-nums;
}
/*
 * Loadpoints
 */
.loadpoints {
	display: flex;
	flex-direction: column;
	gap: 4px;
	margin-top: 10px;
}
.loadpoint-row {
	display: flex;
	align-items: baseline;
	justify-content: space-between;
	gap: 10px;
	width: 100%;
	min-width: 0;
}
.lp-name {
	display: flex;
	align-items: center;
	gap: 6px;
	min-width: 0;
	font-size: 12px;
}
.lp-name svg {
	flex-shrink: 0;
}
.lp-name span {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.lp-power {
	flex-shrink: 0;
	font-size: 12px;
	font-variant-numeric: tabular-nums;
	white-space: nowrap;
}
/*
 * Nested circuit level
 */
.children {
	margin-top: 16px;
}
</style>
