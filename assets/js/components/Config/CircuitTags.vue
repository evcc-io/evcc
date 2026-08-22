<template>
	<div class="circuit-tags" :class="{ 'circuit-tags--child': depth > 0 }">
		<div v-for="node in nodes" :key="node.name" class="node-block">
			<!-- Circuit header -->
			<div class="node-header">
				<span
					class="node-name"
					:class="{ 'node-name--root': depth === 0 }"
				>
					{{ node.name }}
				</span>
			</div>

			<!-- Limits and utilization bars -->
			<div v-if="parts(node).length" class="bars">
				<div
					v-for="part in parts(node)"
					:key="part.unit"
					class="bar-row"
				>
					<!-- Metric unit -->
					<span class="bar-unit">
						{{ part.unit }}
					</span>
					<!-- Utilization bar -->
					<div class="bar-track">
						<div
							class="bar-fill"
							:class="{ warning: part.warning }"
							:style="{ width: barWidth(part.ratio) + '%' }"
						/>
					</div>
					<!-- Current value -->
					<span
						class="bar-value bar-value--current"
						:class="{ warning: part.warning }"
					>
						{{ part.value }}
					</span>
					<!-- Separator -->
					<span
						class="bar-separator"
						:class="{ warning: part.warning }"
					>
						/
					</span>
					<!-- Limit value -->
					<span
						class="bar-value bar-value--limit"
						:class="{ warning: part.warning }"
					>
						{{ part.limit }}
					</span>
					<!-- Value unit -->
					<span
						class="bar-value-unit"
						:class="{ warning: part.warning }"
					>
						{{ part.unit }}
					</span>
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
					<svg
						class="lp-icon"
						width="11"
						height="11"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linejoin="round"
					>
						<path d="M13 2 6 13h5l-1 9 7-11h-5l1-9z" />
					</svg>
					<span class="lp-name evcc-gray">
						{{ lp.title || lp.name }}
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
	value: string;
	limit: string;
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

			// Power is always shown first.
			if (node.maxPower !== undefined) {
				const power = node.power ?? 0;
				const ratio = node.maxPower > 0 ? power / node.maxPower : 0;

				result.push({
					unit: "kW",

					value: this.fmtW(power / 1000, this.POWER_UNIT.KW, false),

					limit: this.fmtW(
						node.maxPower / 1000,
						this.POWER_UNIT.KW,
						false,
					),

					ratio,
					warning: ratio >= 1,
				});
			}

			// Current is always shown after power.
			if (node.maxCurrent !== undefined) {
				const current = node.current ?? 0;
				const ratio =
					node.maxCurrent > 0 ? current / node.maxCurrent : 0;

				result.push({
					unit: "A",

					value: this.fmtW(current, this.POWER_UNIT.W, false),

					limit: this.fmtW(node.maxCurrent, this.POWER_UNIT.W, false),

					ratio,
					warning: ratio >= 1,
				});
			}

			return result;
		},
		barWidth(ratio: number): number {
			return Math.max(0, Math.min(100, ratio * 100));
		},
		loadpointsFor(node: CircuitNode): ConfigLoadpoint[] {
			return this.loadpoints.filter(
				(lp) => lp.circuit === node.title?.toLowerCase(),
			);
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
.circuit-tags--child {
	border-left: 1px solid var(--evcc-gray-10);
	padding-left: 12px;
	box-sizing: border-box;
}
.node-block {
	width: 100%;
	min-width: 0;
	box-sizing: border-box;
}
.node-block + .node-block {
	margin-top: 14px;
}
.node-header {
	width: 100%;
	min-width: 0;
	margin-bottom: 6px;
}
.node-name {
	display: block;
	max-width: 100%;
	min-width: 0;
	font-size: 13px;
	font-weight: 700;
	line-height: 1.2;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.node-name--root {
	font-size: 15px;
}
.bars {
	display: grid;
	grid-template-columns:
		22px
		minmax(30px, 1fr)
		auto
		auto
		auto
		18px;
	align-items: center;
	column-gap: 0;
	row-gap: 5px;
	width: 100%;
	min-width: 0;
}
.bar-row {
	display: contents;
}
.bar-unit {
	padding-right: 6px;
	font-size: 10px;
	font-weight: 700;
	line-height: 1;
	color: var(--evcc-gray);
	text-align: right;
}
.bar-track {
	width: auto;
	min-width: 0;
	height: 4px;
	margin-right: 6px;
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
.bar-value {
	min-width: 0;
	font-size: 11px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;
	white-space: nowrap;
}
.bar-value--current {
	text-align: right;
}
.bar-value--limit {
	text-align: left;
}
.bar-separator {
	font-size: 11px;
	line-height: 1.2;
	text-align: center;
	white-space: nowrap;
}
.bar-value-unit {
	margin-left: 6px;
	font-size: 11px;
	line-height: 1.2;
	text-align: left;
	white-space: nowrap;
}
.warning {
	color: var(--bs-warning);
	font-weight: 700;
}
.bar-fill.warning {
	background: var(--bs-warning);
}
.root-power {
	margin-top: 3px;
	font-size: 12px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;
}
.loadpoints {
	display: flex;
	flex-direction: column;
	gap: 3px;
	margin-top: 7px;
}
.loadpoint-row {
	display: grid;
	grid-template-columns:
		22px
		minmax(30px, 1fr)
		auto
		auto
		auto
		18px;
	align-items: center;
	column-gap: 0;
	width: 100%;
	min-width: 0;
}
.lp-icon {
	grid-column: 1;
	justify-self: end;
	margin-right: 6px;
	color: var(--evcc-gray);
}
.lp-name {
	grid-column: 2;
	min-width: 0;
	font-size: 11px;
	line-height: 1.2;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.lp-power {
	grid-column: 3 / 7;
	justify-self: end;
	font-size: 11px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;
	text-align: right;
	white-space: nowrap;
}
.children {
	margin-top: 11px;
}
</style>
