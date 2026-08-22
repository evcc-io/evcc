<template>
	<div
		class="circuit-tags w-100 min-w-0"
		:class="{ 'circuit-tags--child': depth > 0 }"
	>
		<div
			v-for="node in nodes"
			:key="node.name"
			class="node-block w-100 min-w-0"
		>
			<!-- Circuit header -->
			<div class="node-header w-100 min-w-0">
				<span
					class="node-name d-block mw-100 min-w-0 fw-bold text-truncate"
					:class="{ 'node-name--root': depth === 0 }"
				>
					{{ node.name }}
				</span>
			</div>

			<!-- Measurements and loadpoints share the same grid -->
			<div
				v-if="parts(node).length || loadpointsFor(node).length"
				class="measurement-grid d-grid align-items-center w-100 min-w-0"
			>
				<!-- Limits and utilization bars -->
				<div
					v-for="part in parts(node)"
					:key="part.unit"
					class="bar-row"
				>
					<!-- Metric unit -->
					<span class="bar-unit fw-bold lh-1">
						{{ part.unit }}
					</span>

					<!-- Utilization bar -->
					<div class="bar-track min-w-0 overflow-hidden">
						<div
							class="bar-fill h-100"
							:class="{ warning: part.warning }"
							:style="{ width: barWidth(part.ratio) + '%' }"
						/>
					</div>

					<!-- Current value -->
					<span
						class="bar-value bar-value--current min-w-0 text-nowrap"
						:class="{ warning: part.warning }"
					>
						{{ part.value }}
					</span>

					<!-- Separator -->
					<span
						class="bar-separator text-nowrap"
						:class="{ warning: part.warning }"
					>
						/
					</span>

					<!-- Limit value -->
					<span
						class="bar-value bar-value--limit min-w-0 text-nowrap"
						:class="{ warning: part.warning }"
					>
						{{ part.limit }}
					</span>

					<!-- Value unit -->
					<span class="bar-value-unit text-nowrap">
						{{ part.unit }}
					</span>
				</div>

				<!-- Small visual gap before loadpoints -->
				<div
					v-if="parts(node).length && loadpointsFor(node).length"
					class="loadpoint-spacer"
					aria-hidden="true"
				/>

				<!-- Loadpoints assigned to this circuit -->
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

					<span class="lp-name evcc-gray min-w-0 text-truncate">
						{{ lp.title || lp.name }}
					</span>

					<span class="lp-power-value evcc-gray text-nowrap">
						{{ fmtW(meterPower(lp) / 1000, POWER_UNIT.KW, false) }}
					</span>

					<span class="lp-power-unit evcc-gray text-nowrap">
						kW
					</span>
				</div>
			</div>

			<!-- Root circuit without configured limits -->
			<div
				v-if="!parts(node).length && depth === 0"
				class="root-power evcc-gray"
			>
				{{ fmtW(node.power) }}
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
.min-w-0 {
	min-width: 0;
}

.circuit-tags--child {
	border-left: 1px solid var(--evcc-gray-10);
	padding-left: 12px;
}

.node-block + .node-block {
	margin-top: 14px;
}

.node-header {
	margin-bottom: 6px;
}

.node-name {
	font-size: 13px;
	line-height: 1.2;
}

.node-name--root {
	font-size: 15px;
}

/*
 * One shared grid for both circuit metrics and loadpoints.
 *
 * metric | bar | current | / | limit | spacer | unit
 *
 * Since every row participates in this exact same grid,
 * all numeric values and units are perfectly aligned.
 */
.measurement-grid {
	grid-template-columns:
		22px
		minmax(30px, 1fr)
		auto
		auto
		auto
		6px
		22px;

	column-gap: 0;
	row-gap: 5px;
}

/*
 * Both metric rows and loadpoint rows participate
 * directly in the shared parent grid.
 */
.bar-row,
.loadpoint-row {
	display: contents;
}

/*
 * Center kW/A in the left metric column.
 */
.bar-unit {
	grid-column: 1;

	justify-self: center;

	font-size: 10px;
	text-align: center;

	color: var(--evcc-gray);
}

/*
 * Utilization bar.
 */
.bar-track {
	grid-column: 2;

	height: 4px;
	margin-right: 6px;

	border-radius: 2px;
	background: var(--evcc-gray-10);
}

.bar-fill {
	border-radius: inherit;
	background: var(--evcc-dark-green);

	transition: width 0.2s ease;
}

/*
 * Current value.
 */
.bar-value--current {
	grid-column: 3;

	justify-self: end;

	font-size: 11px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;

	text-align: right;
}

/*
 * Slash separator.
 */
.bar-separator {
	grid-column: 4;

	justify-self: center;

	font-size: 11px;
	line-height: 1.2;

	text-align: center;
}

/*
 * Limit value.
 */
.bar-value--limit {
	grid-column: 5;

	justify-self: start;

	font-size: 11px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;

	text-align: left;
}

/*
 * Circuit value unit.
 *
 * All kW/A values use the exact same shared column.
 */
.bar-value-unit {
	grid-column: 7;

	justify-self: center;

	font-size: 11px;
	line-height: 1.2;

	text-align: center;

	color: inherit;
}

/*
 * Highlight exceeded limits.
 */
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

/*
 * Creates a small gap between circuit metrics
 * and the first loadpoint without breaking alignment.
 */
.loadpoint-spacer {
	grid-column: 1 / -1;
	height: 2px;
}

/*
 * Center the power icon in the same column as kW/A.
 */
.lp-icon {
	grid-column: 1;

	justify-self: center;

	color: var(--evcc-gray);
}

/*
 * Loadpoint name starts exactly where the bar starts.
 */
.lp-name {
	grid-column: 2;

	min-width: 0;

	font-size: 11px;
	line-height: 1.2;
}

/*
 * The loadpoint power spans the complete numeric area
 * and is centered below values such as:
 *
 * 0.0/0.0
 *   0/40
 *   0.0
 */
.lp-power-value {
	grid-column: 3 / 6;

	justify-self: center;

	font-size: 11px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;

	text-align: center;
}

/*
 * The loadpoint kW uses the exact same grid column
 * as the kW/A units above.
 */
.lp-power-unit {
	grid-column: 7;

	justify-self: center;

	font-size: 11px;
	line-height: 1.2;

	text-align: center;
}

.children {
	margin-top: 11px;
}
</style>
