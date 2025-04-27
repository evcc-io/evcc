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
						><small>🧪 {{ solarAdjustText }}</small></label
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
import GenericModal from "../Helper/GenericModal.vue";
import Chart from "./Chart.vue";
import TypeSelect from "./TypeSelect.vue";
import Details from "./Details.vue";
import ActiveSlot from "./ActiveSlot.vue";
import {
	type ForecastSlot,
	type TimeseriesEntry,
	type Forecast,
	ForecastType,
	adjustedSolar,
} from "../../utils/forecast.ts";
import formatter from "../../mixins/formatter.ts";
import settings from "../../settings";
import type { CURRENCY } from "assets/js/types/evcc.ts";
export default defineComponent({
	name: "ForecastModal",
	components: {
		GenericModal,
		ForecastChart: Chart,
		ForecastTypeSelect: TypeSelect,
		ForecastDetails: Details,
		ForecastActiveSlot: ActiveSlot,
	},
	mixins: [formatter],
	props: {
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
		currency: { type: String as PropType<CURRENCY> },
	},
	data(): {
		isModalVisible: boolean;
		selectedType: ForecastType;
		selectedSlot: ForecastSlot | TimeseriesEntry | null;
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
		solarAdjustText() {
			let percent = "";

			const scale = this.forecast.solar?.scale;
			if (scale) {
				const percentDiff = scale * 100 - 100;
				percent = ` (${this.fmtPercentage(percentDiff, 0, true)})`;
			}

			return this.$t("forecast.solarAdjust", { percent });
		},
	},
	watch: {
		isModalVisible(newVal) {
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
		updateSlot(slot: ForecastSlot | TimeseriesEntry | null) {
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
