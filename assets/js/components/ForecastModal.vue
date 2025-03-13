<template>
	<GenericModal
		id="forecastModal"
		:title="$t('forecast.modalTitle')"
		size="xl"
		data-testid="forecast-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<div v-if="isModalVisible">
			<div class="d-flex justify-content-between align-items-center gap-2">
				<ForecastTypeSelect v-model="selectedType" :forecast="forecast" />
				<ForecastActiveSlot :activeSlot="selectedSlot" />
			</div>

			<ForecastChart
				class="my-3"
				:grid="forecast.grid"
				:solar="solar"
				:co2="forecast.co2"
				:currency="currency"
				:selected="selectedType"
				@selected="updateSlot"
			/>
			<ForecastDetails
				:type="selectedType"
				:grid="forecast.grid"
				:solar="solar"
				:co2="forecast.co2"
				:currency="currency"
			/>
			<div v-if="showSolarAdjust" class="form-check form-switch mt-4">
				<input
					id="solarForecastAdjust"
					:checked="solarAdjusted"
					class="form-check-input"
					type="checkbox"
					role="switch"
					@change="changeAdjusted"
				/>
				<div class="form-check-label">
					<label for="solarForecastAdjust"
						><small>🧪 {{ $t("forecast.solarAdjust") }}</small></label
					>
				</div>
			</div>
		</div>
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
import {
	type PriceSlot,
	type TimeseriesEntry,
	type EventEntry,
	type Forecast,
	ForecastType,
	adjustedSolar,
} from "../utils/forecast";
import formatter from "../mixins/formatter";
import settings from "../settings";
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
		selectedSlot: PriceSlot | TimeseriesEntry | EventEntry | null;
	} {
		return {
			isModalVisible: false,
			selectedType: ForecastType.Solar,
			selectedSlot: null,
		};
	},
	computed: {
		solarAdjusted() {
			return settings.solarAdjusted;
		},
		showSolarAdjust() {
			return !!this.forecast.solar && this.$hiddenFeatures();
		},
		solar() {
			return this.showSolarAdjust && this.solarAdjusted
				? adjustedSolar(this.forecast.solar)
				: this.forecast.solar;
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
		updateSlot(slot: PriceSlot | TimeseriesEntry | EventEntry | null) {
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
		changeAdjusted() {
			settings.solarAdjusted = !settings.solarAdjusted;
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
