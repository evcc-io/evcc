<template>
	<GenericModal
		id="forecastModal"
		:title="$t('forecast.modalTitle')"
		size="xl"
		data-testid="forecast-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<div class="d-flex justify-content-between align-items-center">
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
			<div v-if="activeSlotName">
				{{ activeSlotName }}
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
			selectedType: ForecastType.Price,
			types: Object.values(ForecastType),
			selectedSlot: null,
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
				const day = this.weekdayShort(startDate);
				const range = `${startDate.getHours()}–${endDate.getHours()}`;
				return this.$t("main.targetChargePlan.timeRange", { day, range });
			}
			return null;
		},
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
	},
});
</script>

<style scoped></style>
