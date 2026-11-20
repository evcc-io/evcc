<template>
	<IconSelectGroup>
		<IconSelectItem
			v-for="option in options"
			:key="option.value"
			:active="chart === option.value"
			:title="$t(`energy.consumers.chart.${option.value}`)"
			@click="$emit('update:chart', option.value)"
		>
			<component :is="option.icon" />
		</IconSelectItem>
	</IconSelectGroup>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import IconSelectGroup from "../Helper/IconSelectGroup.vue";
import IconSelectItem from "../Helper/IconSelectItem.vue";
import BarChartIcon from "../MaterialIcon/BarChart.vue";
import FullStackedBarChartIcon from "../MaterialIcon/FullStackedBarChart.vue";
import { ENTITY_CHART } from "./types";

const OPTIONS = [
	{ value: ENTITY_CHART.BARS, icon: BarChartIcon },
	{ value: ENTITY_CHART.PATTERN, icon: FullStackedBarChartIcon },
];

export default defineComponent({
	name: "EntityChartToggle",
	components: { IconSelectGroup, IconSelectItem },
	props: {
		chart: { type: String as PropType<ENTITY_CHART>, default: ENTITY_CHART.BARS },
	},
	emits: ["update:chart"],
	computed: {
		options() {
			return OPTIONS;
		},
	},
});
</script>
