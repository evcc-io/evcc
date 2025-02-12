<template>
	<GenericModal
		id="forecastModal"
		:title="$t('forecast.modalTitle')"
		size="xl"
		data-testid="forecast-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<div class="d-flex justify-content-start">
			<IconSelectGroup>
				<IconSelectItem
					v-for="type in types"
					:key="type"
					:active="selectedType === type"
					:label="$t(`forecast.type.${type}`)"
					hideLabelOnMobile
					@click="updateType(type)"
				>
					<component :is="typeIcons[type]"></component>
				</IconSelectItem>
			</IconSelectGroup>
		</div>

		<ForecastChart
			class="my-3"
			:grid="forecast.grid"
			:solar="forecast.solar"
			:co2="forecast.co2"
			:currency="currency"
			:selected="selectedType"
		/>
	</GenericModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/sun";
import { defineComponent } from "vue";
import type { PropType } from "vue";
import GenericModal from "./GenericModal.vue";
import ForecastChart from "./ForecastChart.vue";
import IconSelectItem from "./IconSelectItem.vue";
import IconSelectGroup from "./IconSelectGroup.vue";
import DynamicPriceIcon from "./MaterialIcon/DynamicPrice.vue";
import { type PriceSlot } from "../utils/forecast";
import formatter from "../mixins/formatter";

interface Forecast {
	grid?: PriceSlot[];
	solar?: PriceSlot[];
	co2?: PriceSlot[];
}

export const TYPES = {
	SOLAR: "solar",
	PRICE: "price",
	CO2: "co2",
};

export default defineComponent({
	name: "ForecastModal",
	components: { GenericModal, ForecastChart, IconSelectItem, IconSelectGroup },
	mixins: [formatter],
	props: {
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
		currency: { type: String },
	},
	data: function () {
		return {
			isModalVisible: false,
			selectedType: TYPES.PRICE,
			types: Object.values(TYPES),
		};
	},
	computed: {
		solarAvailable() {
			return !!this.forecast.solar;
		},
		solarSlots() {
			return this.forecast?.solar || [];
		},
		typeIcons() {
			return {
				[TYPES.SOLAR]: "shopicon-regular-sun",
				[TYPES.PRICE]: DynamicPriceIcon,
				[TYPES.CO2]: "shopicon-regular-eco1",
			};
		},
	},
	methods: {
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
		updateType: function (type: string) {
			this.selectedType = type;
		},
	},
});
</script>

<style scoped></style>
