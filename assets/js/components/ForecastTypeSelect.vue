<template>
	<IconSelectGroup>
		<IconSelectItem
			v-for="type in types"
			:key="type"
			:active="selectedType === type"
			:label="$t(`forecast.type.${type}`)"
			:disabled="!availableTypes[type]"
			hideLabelOnMobile
			@click="$emit('update:modelValue', type)"
		>
			<component :is="typeIcons[type]"></component>
		</IconSelectItem>
	</IconSelectGroup>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/sun";
import { defineComponent } from "vue";
import IconSelectItem from "./IconSelectItem.vue";
import IconSelectGroup from "./IconSelectGroup.vue";
import DynamicPriceIcon from "./MaterialIcon/DynamicPrice.vue";
import { ForecastType } from "../utils/forecast";

export default defineComponent({
	name: "ForecastTypeSelect",
	components: { IconSelectItem, IconSelectGroup },
	props: {
		modelValue: { type: String as () => ForecastType, required: true },
		forecast: { type: Object, required: true },
	},
	emits: ["update:modelValue"],
	computed: {
		selectedType() {
			return this.modelValue;
		},
		types() {
			return Object.values(ForecastType);
		},
		availableTypes() {
			return {
				[ForecastType.Solar]: !!this.forecast.solar,
				[ForecastType.Price]: !!this.forecast.grid,
				[ForecastType.Co2]: !!this.forecast.co2,
			};
		},
		typeIcons() {
			return {
				[ForecastType.Solar]: "shopicon-regular-sun",
				[ForecastType.Price]: DynamicPriceIcon,
				[ForecastType.Co2]: "shopicon-regular-eco1",
			};
		},
	},
});
</script>
