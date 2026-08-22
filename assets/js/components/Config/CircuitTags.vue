<template>
	<div class="circuit-tags" :class="{ 'circuit-tags--child': depth > 0 }">
		<div v-for="node in nodes" :key="node.name" class="node-block">
			<!-- Circuit header with limits -->
			<div class="node-header">
				<span
					class="node-name"
					:class="{ 'node-name--root': depth === 0 }"
				>
					{{ node.name }}
				</span>

				<div v-if="parts(node).length" class="limit-values">
					<span
						v-for="part in parts(node)"
						:key="part.unit"
						class="limit-value"
						:class="{ warning: part.warning }"
					>
						{{ part.label }}
					</span>
				</div>
			</div>

			<!-- Progress bars -->
			<div v-if="parts(node).length" class="bars">
				<div
					v-for="part in parts(node)"
					:key="part.unit"
					class="bar-row"
				>
					<span class="bar-unit">
						{{ part.unit }}
					</span>

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

			// Power is always added first.
			if (node.maxPower !== undefined) {
				const power = node.power ?? 0;
				const ratio = node.maxPower > 0 ? power / node.maxPower : 0;

				result.push({
					unit: "kW",
					label: `${this.fmtW(
						power / 1000,
						this.POWER_UNIT.KW,
						false,
					)} / ${this.fmtW(node.maxPower / 1000)}`,
					ratio,
					warning: ratio >= 1,
				});
			}

			// Current is always added after power.
			if (node.maxCurrent !== undefined) {
				const current = node.current ?? 0;
				const ratio =
					node.maxCurrent > 0 ? current / node.maxCurrent : 0;

				result.push({
					unit: "A",
					label: `${this.fmtW(
						current,
						this.POWER_UNIT.W,
						false,
					)} / ${this.fmtW(
						node.maxCurrent,
						this.POWER_UNIT.W,
						false,
					)} A`,
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

/*
 * Continuous hierarchy line for nested circuits.
 * A small indentation preserves as much horizontal space as possible.
 */
.circuit-tags--child {
	border-left: 1px solid var(--evcc-gray-10);
	padding-left: 8px;
}

.node-block {
	width: 100%;
	min-width: 0;
	box-sizing: border-box;
}

/*
 * Compact spacing between sibling circuits.
 * The hierarchy line stays continuous because it belongs
 * to the shared sibling container.
 */
.node-block + .node-block {
	margin-top: 12px;
}

/*
 * Circuit name and limit values share the same row whenever
 * enough horizontal space is available.
 *
 * If the content becomes too wide, the complete limit block
 * automatically wraps below the circuit name.
 */
.node-header {
	display: flex;
	flex-wrap: wrap;
	align-items: baseline;
	justify-content: space-between;

	column-gap: 12px;
	row-gap: 3px;

	width: 100%;
	min-width: 0;
}

.node-name {
	flex: 0 1 auto;
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

/*
 * Keep power and current values together as one block.
 * The complete block is aligned to the right.
 */
.limit-values {
	display: flex;
	align-items: baseline;
	justify-content: flex-end;

	gap: 10px;

	margin-left: auto;

	font-size: 12px;
	line-height: 1.25;
	font-variant-numeric: tabular-nums;

	white-space: nowrap;
}

.limit-value {
	flex-shrink: 0;
	white-space: nowrap;
}

/*
 * Progress bars.
 */
.bars {
	display: flex;
	flex-direction: column;

	gap: 3px;

	width: 100%;
	margin-top: 4px;
}

.bar-row {
	display: flex;
	align-items: center;

	gap: 5px;

	width: 100%;
	min-width: 0;
}

/*
 * Unit labels use a fixed width so both bars start
 * at exactly the same horizontal position.
 */
.bar-unit {
	width: 18px;
	flex: 0 0 18px;

	font-size: 9px;
	font-weight: 700;
	line-height: 1;

	color: var(--evcc-gray);

	text-align: right;
}

.bar-track {
	flex: 1 1 auto;

	min-width: 0;
	height: 3px;

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
 * Root circuit without configured limits.
 */
.root-power {
	margin-top: 3px;

	font-size: 12px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;
}

/*
 * Loadpoints assigned directly to the current circuit.
 */
.loadpoints {
	display: flex;
	flex-direction: column;

	gap: 2px;

	margin-top: 6px;
}

.loadpoint-row {
	display: flex;
	align-items: baseline;
	justify-content: space-between;

	gap: 6px;

	width: 100%;
	min-width: 0;
}

.lp-name {
	display: flex;
	align-items: center;

	gap: 4px;

	min-width: 0;

	font-size: 11px;
	line-height: 1.2;
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

	font-size: 11px;
	line-height: 1.2;
	font-variant-numeric: tabular-nums;

	white-space: nowrap;
}

/*
 * Nested circuit level.
 */
.children {
	margin-top: 9px;
}
</style>
