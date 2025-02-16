<template>
	<GenericModal
		id="forecastModal"
		:title="$t('forecast.modalTitle')"
		size="xl"
		data-testid="forecast-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<div v-if="isModalVisible" class="d-flex justify-content-between align-items-center gap-2">
			<ForecastTypeSelect v-model="selectedType" :forecast="forecast" />
			<ForecastActiveSlot :activeSlot="selectedSlot" />
		</div>

		<ForecastChart
			class="my-3"
			:grid="forecast.grid"
			:solar="forecast.solar"
			:co2="forecast.co2"
			:currency="currency"
			:selected="selectedType"
			@selected="updateSlot"
		/>

		<ForecastDetails :type="selectedType" :slots="selectedSlots" :currency="currency" />
	</GenericModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/sun";
import { defineComponent } from "vue";
import type { PropType } from "vue";
import GenericModal from "./GenericModal.vue";
import ForecastChart from "./ForecastChart.vue";
import ForecastTypeSelect from "./ForecastTypeSelect.vue";
import ForecastDetails from "./ForecastDetails.vue";
import ForecastActiveSlot from "./ForecastActiveSlot.vue";
import { type PriceSlot, ForecastType } from "../utils/forecast";
import formatter from "../mixins/formatter";

interface Forecast {
	grid?: PriceSlot[];
	solar?: PriceSlot[];
	co2?: PriceSlot[];
}

export default defineComponent({
	name: "ForecastModal",
	components: {
		GenericModal,
		ForecastChart,
		ForecastTypeSelect,
		ForecastDetails,
		ForecastActiveSlot,
	},
	mixins: [formatter],
	props: {
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
		currency: { type: String },
	},
	data: function (): {
		isModalVisible: boolean;
		selectedType: ForecastType;
		selectedSlot: PriceSlot | null;
	} {
		return {
			isModalVisible: false,
			selectedType: ForecastType.Solar,
			selectedSlot: null,
		};
	},
	computed: {
		selectedSlots() {
			const slots = {
				[ForecastType.Solar]: this.forecast.solar,
				[ForecastType.Price]: this.forecast.grid,
				[ForecastType.Co2]: this.forecast.co2,
			};
			return slots[this.selectedType] || [];
		},
	},
	watch: {
		isModalVisible: function (newVal) {
			if (newVal) {
				this.updateSelectedType();
			}
		},
	},
	mounted() {
		this.updateSelectedType();
	},
	methods: {
		modalVisible() {
			this.isModalVisible = true;
		},
		modalInvisible() {
			this.isModalVisible = false;
		},
		updateSlot(slot: PriceSlot | null) {
			this.selectedSlot = slot;
		},
		updateSelectedType() {
			const availableTypes = {
				[ForecastType.Solar]: !!this.forecast.solar,
				[ForecastType.Price]: !!this.forecast.grid,
				[ForecastType.Co2]: !!this.forecast.co2,
			};

			// selection has data, do nothing
			if (availableTypes[this.selectedType]) return;
			// fallback to first available type
			this.selectedType =
				Object.values(ForecastType).find((type) => availableTypes[type]) ||
				Object.values(ForecastType)[0];
		},
	},
});
</script>

<style scoped>
.value {
	font-size: 18px;
	font-weight: bold;
}
.extraValue {
	color: var(--evcc-gray);
	font-size: 14px;
	font-weight: normal;
}
.label {
	color: var(--evcc-gray);
	text-transform: uppercase;
}
</style>
