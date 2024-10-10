<template>
	<ul
		class="p-0 d-flex flex-wrap column-gap-4 row-gap-2 justify-content-start overflow-hidden"
		:class="extraClass"
	>
		<li
			class="legend-item d-flex align-items-baseline gap-2 no-wrap overflow-hidden"
			:class="{ 'legend-item---grid': grid }"
			v-for="legend in legends"
			:key="legend.label"
		>
			<div
				v-if="legend.color"
				class="legend-color align-self-center me-1"
				:style="{ backgroundColor: legend.color }"
			></div>
			<div class="text-truncate flex-shrink-1">{{ legend.label }}</div>
			<div v-if="legend.value" class="text-muted text-nowrap">{{ legend.value }}</div>
		</li>
	</ul>
</template>

<script>
export default {
	name: "LegendList",
	props: {
		legends: Array,
		extraClass: { type: [String, Array, Object], default: "" },
		grid: Boolean,
	},
};
</script>

<style scoped>
.legend-color {
	width: 1rem;
	height: 1rem;
	border-radius: 50%;
}
.legend-item---grid {
	justify-content: space-between;
	width: 100px;
}
</style>
