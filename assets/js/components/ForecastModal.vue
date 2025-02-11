<template>
	<GenericModal
		id="forecastModal"
		:title="$t('forecast.modalTitle')"
		size="xl"
		data-testid="forecast-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<ForecastChart :grid="forecast.grid" :solar="forecast.solar" :co2="forecast.co2" />

		<h6>Solar</h6>
		<p>{{ solarToday }}<br />{{ solarTomorrow }}</p>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import type { PropType } from "vue";
import GenericModal from "./GenericModal.vue";
import ForecastChart from "./ForecastChart.vue";
import { todaysEnergy, tomorrowsEnergy, type PriceSlot } from "../utils/forecast";
import formatter, { POWER_UNIT } from "../mixins/formatter";

interface Forecast {
	grid?: PriceSlot[];
	solar?: PriceSlot[];
	co2?: PriceSlot[];
}

export default defineComponent({
	name: "ForecastModal",
	components: { GenericModal, ForecastChart },
	mixins: [formatter],
	props: {
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
		currency: { type: String },
	},
	data: function () {
		return {
			isModalVisible: false,
		};
	},
	computed: {
		solarAvailable() {
			return !!this.forecast.solar;
		},
		solarSlots() {
			return this.forecast?.solar || [];
		},
		solarToday() {
			const energy = this.fmtWh(todaysEnergy(this.solarSlots), POWER_UNIT.KW);
			return `${energy} remaining today`;
		},
		solarTomorrow() {
			const energy = this.fmtWh(tomorrowsEnergy(this.solarSlots), POWER_UNIT.KW);
			return `${energy} tomorrow`;
		},
	},
	methods: {
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
	},
});
</script>

<style scoped></style>
