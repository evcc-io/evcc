<template>
	<ul
		class="root p-0 d-flex flex-wrap column-gap-4 row-gap-2 overflow-hidden"
		:class="{
			'root--small-equal-widths': smallEqualWidths,
			'root--grid': grid,
		}"
	>
		<li
			v-for="legend in legends"
			:key="legend.label"
			class="legend-item d-flex align-items-baseline gap-2 no-wrap overflow-hidden"
		>
			<div
				v-if="legend.color"
				class="legend-color align-self-center me-1"
				:style="{ backgroundColor: legend.color }"
			></div>
			<div class="legend-label text-nowrap">{{ legend.label }}</div>
			<div
				v-for="value in valueList(legend.value)"
				:key="value"
				class="text-muted text-nowrap legend-value text-end"
			>
				{{ value }}
			</div>
		</li>
	</ul>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { Legend } from "./types";

export default defineComponent({
	name: "LegendList",
	props: {
		legends: Array as PropType<Legend[]>,
		grid: Boolean,
		smallEqualWidths: Boolean,
	},
	methods: {
		valueList(value: Legend["value"]) {
			if (!value) return [];
			return Array.isArray(value) ? value : [value];
		},
	},
});
</script>

<style scoped>
.root {
	justify-content: flex-start;
}
.legend-color {
	width: 1rem;
	height: 1rem;
	flex-shrink: 0;
	border-radius: 50%;
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
