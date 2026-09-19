<template>
	<div class="d-flex flex-column gap-4 w-100 min-w-0" :class="{ 'ps-4': depth > 0 }">
		<div v-for="node in nodes" :key="node.name" class="w-100 min-w-0">
			<div class="d-flex align-items-center gap-1 min-w-0 mb-1 lh-sm">
				<SubdirectoryArrowRight
					v-if="depth > 0"
					class="child-icon flex-shrink-0"
					:size="ICON_SIZE.XS"
				/>
				<span class="flex-grow-1 min-w-0 fw-bold text-truncate">
					{{ node.title }}
				</span>
			</div>
			<div
				v-if="parts(node).length || loadpointsFor(node).length"
				class="measurement-grid d-grid align-items-center w-100 min-w-0 small lh-sm"
			>
				<div v-for="part in parts(node)" :key="part.unit" class="bar-row evcc-gray">
					<span
						class="bar-value text-nowrap tabular me-2"
						:class="{ 'text-warning': overLimit(part) }"
					>
						{{ part.value }}
					</span>
					<div class="bar-track min-w-0 overflow-hidden">
						<div
							class="bar-fill h-100"
							:class="{ 'bg-warning': overLimit(part) }"
							:style="{ width: barWidth(part.ratio) + '%' }"
						/>
					</div>
					<span class="bar-limit text-nowrap tabular ms-2">
						{{ part.limit ?? "__" }}
					</span>
					<span class="bar-unit text-nowrap ms-1">{{ part.unit }}</span>
				</div>
				<div
					v-for="lp in loadpointsFor(node)"
					:key="lp.name"
					class="loadpoint-row evcc-gray"
				>
					<span class="lp-name d-flex align-items-center gap-1 min-w-0">
						<SubdirectoryArrowRight class="flex-shrink-0" :size="ICON_SIZE.XS" />
						<span class="text-truncate">{{ lp.title || lp.name }}</span>
					</span>
					<span class="lp-power text-nowrap tabular ms-2">
						{{ fmtW(lp.power, POWER_UNIT.KW, false) }}
					</span>
					<span class="bar-unit text-nowrap ms-1">kW</span>
				</div>
			</div>
			<div v-if="node.children?.length" class="mt-3">
				<CircuitTags :nodes="node.children" :loadpoints="loadpoints" :depth="depth + 1" />
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import formatter from "@/mixins/formatter.ts";
import type { CircuitNode } from "@/utils/circuits.ts";
import { ICON_SIZE } from "@/types/evcc";
import SubdirectoryArrowRight from "../MaterialIcon/SubdirectoryArrowRight.vue";

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
}

export default {
	name: "CircuitTags",
	components: { SubdirectoryArrowRight },
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
	data() {
		return { ICON_SIZE };
	},
	methods: {
		parts(node: CircuitNode): LimitPart[] {
			const result: LimitPart[] = [];
			if (node.maxCurrent !== undefined) {
				const current = node.current ?? 0;
				const ratio = node.maxCurrent > 0 ? current / node.maxCurrent : 0;
				result.push({
					unit: "A",
					value: this.fmtNumber(current, 1),
					limit: this.fmtNumberToLocale(node.maxCurrent),
					ratio,
				});
			}
			// power always shown, without limit as full bar
			const power = node.power ?? 0;
			const powerRatio = node.maxPower ? power / node.maxPower : 1;
			result.push({
				unit: "kW",
				value: this.fmtW(power, this.POWER_UNIT.KW, false),
				limit: node.maxPower
					? this.fmtW(node.maxPower, this.POWER_UNIT.KW, false)
					: undefined,
				ratio: powerRatio,
			});
			return result;
		},
		overLimit(part: LimitPart): boolean {
			return part.limit !== undefined && part.ratio >= 1;
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
.measurement-grid {
	/* limit column fits "88.8", grows for larger values at the bar's expense */
	grid-template-columns: minmax(3.5ch, auto) minmax(0, 1fr) minmax(3.5ch, auto) auto;
	row-gap: 0.375rem;
}
.bar-row,
.loadpoint-row {
	display: contents;
}
.bar-value {
	grid-column: 1;
	justify-self: end;
}
.bar-track {
	grid-column: 2;
	height: 4px;
	border-radius: 2px;
	background: var(--evcc-gray-10);
}
.bar-limit,
.lp-power {
	grid-column: 3;
	justify-self: end;
}
.bar-unit {
	grid-column: 4;
}
.bar-fill {
	border-radius: inherit;
	background-color: var(--evcc-dark-green);
	transition: width 0.2s ease;
}
/* icon sits in the parent's gutter, the inset equals icon plus gap */
.child-icon {
	margin-left: -1.5rem;
}
.lp-name {
	grid-column: 2;
}
</style>
