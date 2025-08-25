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
				:feedin="forecast.feedin"
				:currency="currency"
				:selected="selectedType"
				:feedInDisabledZones="feedInDisabledZones"
				@selected="updateSlot"
			/>
			<ForecastDetails
				:type="selectedType"
				:grid="forecast.grid"
				:solar="solar"
				:co2="forecast.co2"
				:feedin="forecast.feedin"
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
			<div v-if="showSmartFeedIn" class="form-check form-switch mt-4">
				<input
					id="solarForecastAdjust"
					:checked="smartFeedInDisableLimit !== null"
					class="form-check-input"
					type="checkbox"
					role="switch"
					@change="toggleSmartFeedInDisableLimit"
				/>
				<div class="form-check-label">
					<label for="smartFeedInDisableLimit">
						<i18n-t
							keypath="forecast.smartFeedInDisable"
							tag="small"
							class="d-inline"
							scope="global"
						>
							<template #limit>
								<CustomSelect
									v-if="smartFeedInDisableLimit !== null"
									id="smartFeedInDisableLimit"
									class="custom-select-inline"
									:options="smartFeedInDisableLimitOptions"
									:selected="smartFeedInDisableLimit"
									@change="changeSmartFeedInDisableLimit"
								>
									<span class="text-decoration-underline">
										≤
										{{
											fmtPricePerKWh(smartFeedInDisableLimit, currency, true)
										}}
									</span>
								</CustomSelect>
								<span v-else>{{ $t("forecast.smartFeedInDisableLow") }}</span>
							</template>
						</i18n-t>
						<FeedInPatternIndicator
							v-if="smartFeedInDisableLimit !== null"
							class="ms-2"
							:title="$t('forecast.smartFeedInDisabledZones')"
						/>
					</label>
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
import CustomSelect from "../Helper/CustomSelect.vue";
import FeedInPatternIndicator from "./FeedInPatternIndicator.vue";

import formatter from "@/mixins/formatter";
import settings from "@/settings";
import type { CURRENCY, Forecast } from "@/types/evcc";
import { ForecastType, adjustedSolar } from "@/utils/forecast";
import type { ForecastSlot, TimeseriesEntry, ForecastZone } from "./types";
import api from "@/api";
export default defineComponent({
	name: "ForecastModal",
	components: {
		GenericModal,
		ForecastChart: Chart,
		ForecastTypeSelect: TypeSelect,
		ForecastDetails: Details,
		ForecastActiveSlot: ActiveSlot,
		CustomSelect,
		FeedInPatternIndicator,
	},
	mixins: [formatter],
	props: {
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
		currency: { type: String as PropType<CURRENCY> },
		smartFeedInDisableLimit: {
			type: [Number, null] as PropType<number | null>,
			default: null,
		},
		smartFeedInDisableAvailable: { type: Boolean, default: false },
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
			return (
				!!this.forecast.solar &&
				this.selectedType === ForecastType.Solar &&
				this.$hiddenFeatures()
			);
		},
		showSmartFeedIn() {
			return (
				this.smartFeedInDisableAvailable &&
				this.selectedType === ForecastType.FeedIn &&
				this.$hiddenFeatures()
			);
		},
		solar() {
			return this.showSolarAdjust && this.solarAdjusted
				? adjustedSolar(this.forecast.solar)
				: this.forecast.solar;
		},
		feedInDisabledZones(): ForecastZone[] {
			const zones: ForecastZone[] = [];

			// Only calculate zones if limit is set and feedin data exists
			if (
				this.smartFeedInDisableLimit === null ||
				!this.forecast.feedin ||
				this.forecast.feedin.length === 0
			) {
				return zones;
			}

			// Group consecutive slots that are below the limit
			let currentZone: ForecastZone | null = null;

			this.forecast.feedin.forEach((slot) => {
				if (
					this.smartFeedInDisableLimit !== null &&
					slot.value <= this.smartFeedInDisableLimit
				) {
					if (!currentZone) {
						currentZone = { start: slot.start, end: slot.end };
					} else {
						currentZone.end = slot.end;
					}
				} else {
					if (currentZone) {
						zones.push(currentZone);
						currentZone = null;
					}
				}
			});

			// Don't forget the last zone if it exists
			if (currentZone) {
				zones.push(currentZone);
			}

			return zones;
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
		smartFeedInDisableLimitOptions() {
			const options = [];
			for (let i = 0; i < 100; i++) {
				options.push({
					value: (i - 50) / 100,
					name: this.fmtPricePerKWh(i, this.currency, true),
				});
			}
			return options;
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
				[ForecastType.FeedIn]: !!this.forecast.feedin,
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
		toggleSmartFeedInDisableLimit() {
			if (this.smartFeedInDisableLimit === null) {
				api.post("smartfeedindisablelimit/0");
			} else {
				api.delete("smartfeedindisablelimit");
			}
		},
		async changeSmartFeedInDisableLimit(e: Event) {
			const limit = (e.target as HTMLSelectElement).value || 0;
			await api.post(`smartfeedindisablelimit/${encodeURIComponent(limit)}`);
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
