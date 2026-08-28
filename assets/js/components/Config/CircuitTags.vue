<template>
	<div class="w-100 min-w-0" :class="{ 'ps-3': depth > 0 }">
		<div v-for="node in nodes" :key="node.name" class="node-block w-100 min-w-0">
			<div class="w-100 min-w-0 mb-1">
				<div class="node-header-row d-flex flex-wrap align-items-baseline min-w-0">
					<span
						class="node-name d-block mw-100 min-w-0 fw-bold text-truncate"
						:class="{ 'node-name--root': depth === 0 }"
					>
						{{ node.name || node.title }}
					</span>
					<div
						v-if="parts(node).length"
						class="node-header-values d-flex flex-shrink-0 ms-auto"
					>
						<span
							v-for="part in parts(node)"
							:key="part.unit"
							class="text-nowrap"
							:class="{ 'text-warning fw-bold': part.warning }"
						>
							{{ part.value }}/{{ part.limit }} {{ part.unit }}
						</span>
					</div>
				</div>
			</div>
			<div
				v-if="parts(node).length || loadpointsFor(node).length"
				class="measurement-grid d-grid align-items-center w-100 min-w-0"
			>
				<div v-for="part in parts(node)" :key="part.unit" class="bar-row">
					<span class="bar-unit fw-bold lh-1">
						{{ part.unit }}
					</span>
					<div class="bar-track min-w-0 overflow-hidden">
						<div
							class="bar-fill h-100"
							:class="{ 'bg-warning': part.warning }"
							:style="{ width: barWidth(part.ratio) + '%' }"
						/>
					</div>
				</div>
				<div
					v-if="parts(node).length && loadpointsFor(node).length"
					class="loadpoint-spacer"
					aria-hidden="true"
				/>
				<div v-for="lp in loadpointsFor(node)" :key="lp.name" class="loadpoint-row">
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
					<span class="lp-name evcc-gray min-w-0 text-truncate">
						{{ lp.title || lp.name }}
					</span>
					<span class="lp-power-value evcc-gray text-nowrap">
						{{ fmtW(meterPower(lp), POWER_UNIT.KW, false) }}
					</span>
					<span class="lp-power-unit evcc-gray text-nowrap"> kW </span>
				</div>
			</div>
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
			if (node.maxPower !== undefined) {
				const power = node.power ?? 0;
				const ratio = node.maxPower > 0 ? power / node.maxPower : 0;
				result.push({
					unit: "kW",
					value: this.fmtW(power, this.POWER_UNIT.KW, false),
					limit: this.fmtW(node.maxPower, this.POWER_UNIT.KW, false),
					ratio,
					warning: ratio >= 1,
				});
			}
			if (node.maxCurrent !== undefined) {
				const current = node.current ?? 0;
				const ratio = node.maxCurrent > 0 ? current / node.maxCurrent : 0;
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
			return this.loadpoints.filter((lp) => lp.circuit === node.title?.toLowerCase());
		},
		meterPower(lp: ConfigLoadpoint): number {
			const meter = this.meters.find((m) => m.name === lp.meter);
			return meter?.power ?? 0;
		},
	},
};
</script>

<style scoped>
.min-w-0 {
	min-width: 0;
}
.node-block + .node-block {
	margin-top: 14px;
}
.node-header-row {
	gap: 4px 12px;
}
.node-name {
	flex: 1 1 auto;
	font-size: 13px;
	line-height: 1.2;
}
.node-name--root {
	font-size: 15px;
}
.node-header-values {
	gap: 12px;
	font-size: 11px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;
}
.measurement-grid {
	grid-template-columns:
		22px
		minmax(30px, 1fr)
		auto
		auto
		auto
		6px
		22px;
}
.bar-row,
.loadpoint-row {
	display: contents;
}
.bar-unit {
	grid-column: 1;
	padding-right: 6px;
	font-size: 10px;
	color: var(--evcc-gray);
}
.bar-track {
	grid-column: 2 / -1;
	height: 4px;
	border-radius: 2px;
	background: var(--evcc-gray-10);
}
.bar-fill {
	border-radius: inherit;
	background: var(--evcc-dark-green);
	transition: width 0.2s ease;
}
.loadpoint-spacer {
	grid-column: 1 / -1;
	height: 2px;
}
.lp-icon {
	grid-column: 1;
	justify-self: center;
	color: var(--evcc-gray);
}
.lp-name {
	grid-column: 2;
	font-size: 11px;
	line-height: 1.2;
}
.lp-icon,
.lp-name {
	transform: translateX(30px);
}
.lp-power-value {
	grid-column: 3 / 6;
	justify-self: center;
	font-size: 11px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;
}
.lp-power-unit {
	grid-column: 7;
	justify-self: center;
	font-size: 11px;
	line-height: 1.2;
}
.children {
	margin-top: 11px;
}
</style>
