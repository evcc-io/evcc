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
			<IconSelectGroup>
				<IconSelectItem
					v-for="type in types"
					:key="type"
					:active="selectedType === type"
					:label="$t(`forecast.type.${type}`)"
					:disabled="!availableTypes[type]"
					hideLabelOnMobile
					@click="updateType(type)"
				>
					<component :is="typeIcons[type]"></component>
				</IconSelectItem>
			</IconSelectGroup>
			<div v-if="activeSlotName" class="text-end">
				<span class="text-nowrap">{{ activeSlotName.day }} {{ activeSlotName.start }}</span
				>{{ " " }}<span class="text-nowrap">– {{ activeSlotName.end }}</span>
			</div>
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
		<div v-if="selectedType === 'solar'" class="row">
			<div class="col-6 col-sm-4 mb-3 d-flex flex-column">
				<div class="label">Heute</div>
				<div class="value align-items-baseline d-flex flex-wrap column-gap-2">
					<div class="text-primary text-nowrap">3,4 kWh</div>
					<div class="extraValue text-nowrap">verbleibend</div>
				</div>
			</div>
			<div
				class="col-6 col-sm-4 mb-3 d-flex flex-column align-items-end align-items-sm-center"
			>
				<div class="label">Morgen</div>
				<div
					class="value align-items-baseline d-flex flex-wrap column-gap-2 justify-content-end justify-content-sm-center"
				>
					<div class="text-primary text-nowrap">4,4 kWh</div>
				</div>
			</div>
			<div
				class="col-6 col-sm-4 mb-3 d-flex flex-column align-items-start align-items-sm-end"
			>
				<div class="label">Übermorgen</div>
				<div
					class="value align-items-baseline d-flex flex-wrap column-gap-2 justify-content-start justify-content-sm-end"
				>
					<div class="text-primary text-nowrap">11,3 kWh</div>
					<div class="extraValue text-nowrap">teilweise</div>
				</div>
			</div>
		</div>
		<div v-else-if="selectedType === 'price'" class="row">
			<div class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column">
				<div class="label">Bereich</div>
				<div class="value text-price text-nowrap">28.7 – 37.2 ct/kWh</div>
			</div>
			<div
				class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column align-items-sm-end align-items-lg-center"
			>
				<div class="label">Durchschnitt</div>
				<div class="value text-price text-nowrap">33.0 ct/kWh</div>
			</div>
			<div
				class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column align-items-sm-start align-items-lg-end"
			>
				<div class="label">Günstigste Stunde</div>
				<div class="value text-price text-nowrap">Thu 4 AM – 5 AM</div>
			</div>
		</div>
		<div v-else-if="selectedType === 'co2'" class="row">
			<div class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column">
				<div class="label">Bereich</div>
				<div class="value text-co2 text-nowrap">117 - 235 g/kWh</div>
			</div>
			<div
				class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column align-items-sm-end align-items-lg-center"
			>
				<div class="label">Durchschnitt</div>
				<div class="value text-co2 text-nowrap">175 g/kWh</div>
			</div>
			<div
				class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column align-items-sm-start align-items-lg-end"
			>
				<div class="label">Günstigste Stunde</div>
				<div class="value text-co2 text-nowrap">Thu 11 AM – 12 PM</div>
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
import IconSelectItem from "./IconSelectItem.vue";
import IconSelectGroup from "./IconSelectGroup.vue";
import DynamicPriceIcon from "./MaterialIcon/DynamicPrice.vue";
import { type PriceSlot, ForecastType } from "../utils/forecast";
import formatter from "../mixins/formatter";

interface Forecast {
	grid?: PriceSlot[];
	solar?: PriceSlot[];
	co2?: PriceSlot[];
}

export default defineComponent({
	name: "ForecastModal",
	components: { GenericModal, ForecastChart, IconSelectItem, IconSelectGroup },
	mixins: [formatter],
	props: {
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
		currency: { type: String },
	},
	data: function (): {
		isModalVisible: boolean;
		selectedType: ForecastType;
		types: ForecastType[];
		selectedSlot: PriceSlot | null;
	} {
		return {
			isModalVisible: false,
			selectedType: ForecastType.Solar,
			types: Object.values(ForecastType),
			selectedSlot: null,
		};
	},
	computed: {
		availableTypes() {
			return {
				[ForecastType.Solar]: !!this.forecast.solar,
				[ForecastType.Price]: !!this.forecast.grid,
				[ForecastType.Co2]: !!this.forecast.co2,
			};
		},
		solarSlots() {
			return this.forecast?.solar || [];
		},
		typeIcons() {
			return {
				[ForecastType.Solar]: "shopicon-regular-sun",
				[ForecastType.Price]: DynamicPriceIcon,
				[ForecastType.Co2]: "shopicon-regular-eco1",
			};
		},
		activeSlotName() {
			if (this.selectedSlot) {
				const { start, end } = this.selectedSlot;
				const startDate = new Date(start);
				const endDate = new Date(end);

				return {
					day: this.weekdayShort(startDate),
					start: this.hourShort(startDate),
					end: this.hourShort(endDate),
				};
			}
			return null;
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
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
		updateType: function (type: ForecastType) {
			this.selectedType = type;
		},
		updateSlot: function (slot: PriceSlot | null) {
			this.selectedSlot = slot;
		},
		updateSelectedType: function () {
			// selection has data, do nothing
			if (this.availableTypes[this.selectedType]) return;
			// fallback to first available type
			this.selectedType =
				this.types.find((type) => this.availableTypes[type]) || this.types[0];
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
