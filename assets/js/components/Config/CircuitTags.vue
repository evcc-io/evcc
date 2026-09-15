<template>
	<div class="d-flex flex-column gap-3 w-100 min-w-0" :class="{ 'ps-3': depth > 0 }">
		<div v-for="node in nodes" :key="node.name" class="w-100 min-w-0">
			<div
				class="d-flex flex-wrap align-items-baseline gap-1 column-gap-3 min-w-0 mb-1 lh-sm"
			>
				<span
					class="flex-grow-1 mw-100 min-w-0 fw-bold text-truncate"
					:class="{ small: depth > 0 }"
				>
					{{ node.title }}
				</span>
				<div class="d-flex flex-shrink-0 gap-3 ms-auto small tabular">
					<span
						v-for="part in parts(node)"
						:key="part.unit"
						class="text-nowrap"
						:class="{ 'text-warning fw-bold': part.warning }"
					>
						{{ part.value }}<template v-if="part.limit">/{{ part.limit }}</template>
						{{ part.unit }}
					</span>
				</div>
			</div>
			<div
				v-if="barParts(node).length || loadpointsFor(node).length"
				class="measurement-grid d-grid align-items-center w-100 min-w-0 small lh-sm"
			>
				<div v-for="part in barParts(node)" :key="part.unit" class="bar-row">
					<span class="bar-unit fw-bold evcc-gray pe-2">
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
					v-if="barParts(node).length && loadpointsFor(node).length"
					class="loadpoint-spacer"
					aria-hidden="true"
				/>
				<div
					v-for="lp in loadpointsFor(node)"
					:key="lp.name"
					class="loadpoint-row evcc-gray"
				>
					<shopicon-regular-lightning
						class="lp-icon"
						size="s"
					></shopicon-regular-lightning>
					<span class="lp-name min-w-0 text-truncate">
						{{ lp.title || lp.name }}
					</span>
					<span class="lp-power-value text-nowrap tabular">
						{{ fmtW(lp.power, POWER_UNIT.KW, false) }}
					</span>
					<span class="lp-power-unit text-nowrap"> kW </span>
				</div>
			</div>
			<div v-if="node.children?.length" class="mt-2">
				<CircuitTags :nodes="node.children" :loadpoints="loadpoints" :depth="depth + 1" />
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import "@h2d2/shopicons/es/regular/lightning";
import formatter from "@/mixins/formatter.ts";
import type { CircuitNode } from "@/utils/circuits.ts";

export interface CircuitLoadpoint {
	name?: string;
	title: string;
	circuit?: string;
	power: number;
}

interface LimitPart {
	unit: "kW" | "A";
	value: string;
	limit?: string;
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
			type: Array as PropType<CircuitLoadpoint[]>,
			required: true,
		},
		depth: {
			type: Number,
			default: 0,
		},
	},
	methods: {
		parts(node: CircuitNode): LimitPart[] {
			// power always shown, limit may be absent (e.g. external limit only)
			const power = node.power ?? 0;
			const powerRatio = node.maxPower ? power / node.maxPower : 0;
			const result: LimitPart[] = [
				{
					unit: "kW",
					value: this.fmtW(power, this.POWER_UNIT.KW, false),
					limit: node.maxPower
						? this.fmtW(node.maxPower, this.POWER_UNIT.KW, false)
						: undefined,
					ratio: powerRatio,
					warning: powerRatio >= 1,
				},
			];
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
		barParts(node: CircuitNode): LimitPart[] {
			return this.parts(node).filter((p) => p.limit !== undefined);
		},
		barWidth(ratio: number): number {
			return Math.max(0, Math.min(100, ratio * 100));
		},
		loadpointsFor(node: CircuitNode): CircuitLoadpoint[] {
			return this.loadpoints.filter((lp) => lp.circuit === node.name);
		},
	},
};
</script>

<style scoped>
.min-w-0 {
	min-width: 0;
}
.measurement-grid {
	grid-template-columns: auto 30px minmax(30px, 1fr) auto 22px;
}
.bar-row,
.loadpoint-row {
	display: contents;
}
.bar-unit {
	grid-column: 1;
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
	grid-column: 2;
	justify-self: center;
}
.lp-name {
	grid-column: 3;
}
.lp-power-value {
	grid-column: 4;
}
.lp-power-unit {
	grid-column: 5;
	justify-self: center;
}
</style>
