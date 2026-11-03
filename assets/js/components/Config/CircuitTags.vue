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
			<div class="measurement-grid d-grid align-items-center w-100 min-w-0 small lh-sm">
				<div
					v-for="part in parts(node)"
					:key="part.unit"
					class="bar-row"
					:class="STATE_TEXT[part.bar.state]"
				>
					<div class="bar-track d-flex min-w-0 me-2">
						<div
							class="bar-fill h-100"
							:class="[
								STATE_FILL[part.bar.state],
								{
									'bar-striped': part.striped,
									'bar-export': part.export,
									'bar-animated': part.striped && part.active,
								},
							]"
							:style="{ width: part.bar.fill + '%' }"
						/>
						<div
							v-if="part.bar.state === 'danger'"
							class="bar-fill bar-excess h-100 flex-grow-1 bg-danger rounded-start-0"
						/>
					</div>
					<span class="bar-label text-nowrap text-end tabular">{{ part.label }}</span>
					<span class="text-nowrap ms-1">{{ part.unit }}</span>
				</div>
				<div
					v-for="lp in loadpointsFor(node)"
					:key="lp.name"
					class="loadpoint-row d-flex align-items-center gap-1 min-w-0 evcc-gray"
				>
					<SubdirectoryArrowRight class="flex-shrink-0" :size="ICON_SIZE.XS" />
					<span class="text-truncate">{{ lp.title || lp.name }}</span>
					<span class="text-nowrap tabular ms-1">{{
						fmtW(lp.power, POWER_UNIT.KW)
					}}</span>
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
import { limitBar, type CircuitNode, type LimitBar, type LimitState } from "@/utils/circuits.ts";
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
	label: string;
	bar: LimitBar;
	striped?: boolean;
	active?: boolean;
	export?: boolean;
}

const STATE_TEXT: Record<LimitState, string> = {
	normal: "evcc-gray",
	warning: "text-warning",
	danger: "text-danger",
};

// the excess segment continues the fill, so their touching ends are square
const STATE_FILL: Record<LimitState, string> = {
	normal: "",
	warning: "bg-warning",
	danger: "bg-warning rounded-end-0",
};

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
		return { ICON_SIZE, STATE_TEXT, STATE_FILL };
	},
	methods: {
		parts(node: CircuitNode): LimitPart[] {
			const result: LimitPart[] = [];
			if (node.maxCurrent !== undefined) {
				const current = node.current ?? 0;
				result.push({
					unit: "A",
					label: `${this.fmtNumber(current, 1)} / ${this.fmtNumber(node.maxCurrent, 1)}`,
					bar: limitBar(current, node.maxCurrent),
				});
			}
			// power always shown, without limit or on feed-in as full striped bar indicating activity
			const power = node.power ?? 0;
			const feedIn = power < 0;
			const powerLabel = this.fmtW(power, this.POWER_UNIT.KW, false);
			result.push({
				unit: "kW",
				label: node.maxPower
					? `${powerLabel} / ${this.fmtW(node.maxPower, this.POWER_UNIT.KW, false)}`
					: powerLabel,
				bar: limitBar(power, feedIn ? undefined : node.maxPower),
				striped: !node.maxPower || feedIn,
				active: power !== 0,
				export: feedIn,
			});
			return result;
		},
		loadpointsFor(node: CircuitNode): CircuitLoadpoint[] {
			return this.loadpoints.filter((lp) => lp.circuit === node.name);
		},
	},
};
</script>

<style scoped>
.measurement-grid {
	grid-template-columns: minmax(0, 1fr) auto auto;
	row-gap: 0.375rem;
}
.bar-row {
	display: contents;
}
.loadpoint-row {
	grid-column: 1 / -1;
}
.bar-track {
	height: 4px;
	border-radius: 2px;
	background: var(--evcc-gray-10);
}
/* fits "00.0 / 00.0" so changing values don't shift the bar */
.bar-label {
	min-width: 9ch;
}
.bar-fill {
	border-radius: inherit;
	background-color: var(--evcc-dark-green);
	transition: width 0.2s ease;
}
.bar-excess {
	margin-left: 1px;
}
.bar-export {
	background-color: var(--evcc-export);
}
/* stripe period matches the bar height so the pattern stays legible on the thin track */
.bar-striped {
	--stripe: 8px;
	background-image: repeating-linear-gradient(
		-45deg,
		rgba(255, 255, 255, 0.4) 0 calc(var(--stripe) / 2),
		transparent calc(var(--stripe) / 2) var(--stripe)
	);
	background-size: calc(var(--stripe) * 1.4142) 100%;
}
.bar-animated {
	animation: stripe-shift 0.6s linear infinite;
}
.bar-animated.bar-export {
	animation-direction: reverse;
}
@keyframes stripe-shift {
	to {
		background-position: calc(var(--stripe) * 1.4142) 0;
	}
}
@media (prefers-reduced-motion: reduce) {
	.bar-animated {
		animation: none;
	}
}
/* icon sits in the parent's gutter, the inset equals icon plus gap */
.child-icon {
	margin-left: -1.5rem;
}
</style>
