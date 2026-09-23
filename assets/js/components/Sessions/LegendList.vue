<template>
	<ul
		class="root p-0 m-0 row-gap-2"
		:class="[
			columns ? 'root--columns' : 'd-flex flex-wrap column-gap-4',
			{
				'root--small-equal-widths': smallEqualWidths,
				'root--grid': grid,
			},
		]"
		:style="columns ? { '--legend-count': legends?.length ?? 0 } : undefined"
	>
		<li
			v-for="legend in legends"
			:key="legend.label"
			class="legend-item align-items-baseline gap-2 no-wrap"
			:class="{
				'd-flex': !columns,
				'legend-item--columns': columns,
				'legend-item--focusable': legend.focusable,
				'legend-item--dim': legend.dim,
			}"
			:role="legend.focusable ? 'button' : undefined"
			:tabindex="legend.focusable ? 0 : undefined"
			@click="legend.focusable && onFocus(legend)"
			@keydown.enter.prevent="legend.focusable && onFocus(legend)"
			@keydown.space.prevent="legend.focusable && onFocus(legend)"
		>
			<DeviceColorDot
				v-if="legend.color"
				class="align-self-center me-1"
				:title="legend.id || ''"
				:color="legend.color"
				:explicit="legend.id ? deviceColors[legend.id] || '' : ''"
				:line="legend.type === 'line'"
			/>
			<div class="legend-label text-nowrap">{{ legend.label }}</div>
			<div
				v-for="value in valueList(legend.value)"
				:key="value"
				class="text-muted text-nowrap legend-value text-end tabular"
			>
				{{ value }}
			</div>
		</li>
	</ul>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { Legend } from "./types";
import type { DeviceColors } from "@/types/evcc";
import DeviceColorDot from "../Helper/DeviceColorDot.vue";

export default defineComponent({
	name: "LegendList",
	components: { DeviceColorDot },
	props: {
		legends: Array as PropType<Legend[]>,
		grid: Boolean,
		// dot, label and values in aligned columns, wrapping into column groups by breakpoint
		columns: Boolean,
		smallEqualWidths: Boolean,
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
	},
	emits: ["focus"],
	methods: {
		valueList(value: Legend["value"]) {
			if (!value) return [];
			return Array.isArray(value) ? value : [value];
		},
		onFocus(legend: Legend) {
			this.$emit("focus", legend);
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

.root {
	justify-content: flex-start;
}
.legend-item--focusable {
	cursor: pointer;
	user-select: none;
	transition: opacity 150ms;
}
.legend-item--dim {
	opacity: 0.35;
}

.legend-label {
	flex-shrink: 0;
	flex-grow: 0;
}

.root--grid .legend-label {
	flex-grow: 1;
	flex-shrink: 1;
	text-overflow: ellipsis;
	overflow: hidden;
}
.root--grid .legend-item {
	flex-grow: 1;
	flex-basis: 100%;
}
.root--grid .legend-value:last-child {
	flex-basis: 3.5rem;
}

/* column groups of dot, label, values; the outer grid spaces the groups */
/* filled column by column, largest entries first */
.root--columns {
	--legend-groups: 1;
	display: grid;
	grid-template-columns: repeat(var(--legend-groups), minmax(0, 1fr));
	grid-template-rows: repeat(round(up, var(--legend-count) / var(--legend-groups)), auto);
	grid-auto-flow: column;
	column-gap: 2rem;
}
.legend-item--columns {
	display: grid;
	grid-template-columns: auto minmax(0, 1fr) minmax(4.75rem, max-content) minmax(
			3.5rem,
			max-content
		);
	column-gap: 0.5rem;
	align-items: baseline;
}
.legend-item--columns .legend-label {
	overflow: hidden;
	text-overflow: ellipsis;
}
@media (--md-and-up) {
	.root--columns {
		--legend-groups: 2;
	}
}
@media (--xl-and-up) {
	.root--columns {
		--legend-groups: 3;
		column-gap: 4rem;
	}
}

.root--small-equal-widths {
	display: flex;
	justify-content: space-evenly;
}
.root--small-equal-widths .legend-item {
	flex-basis: 8rem;
}
.root--small-equal-widths .legend-label {
	flex-grow: 1;
	flex-shrink: 1;
}
</style>
