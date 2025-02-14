<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader title="Energy Overview" />
		<div class="alert alert-light mb-3">This page is work in progress.</div>
		<div class="row">
			<main class="col-12">
				<div v-if="forecastAvailable">
					<h3 class="fw-normal my-4">
						{{ $t("forecast.modalTitle") }}
					</h3>
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
				</div>
				<div v-else>
					<p>nothing to see here</p>
				</div>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import TopHeader from "../components/TopHeader.vue";
import ForecastChart from "../components/ForecastChart.vue";
import IconSelectItem from "../components/IconSelectItem.vue";
import IconSelectGroup from "../components/IconSelectGroup.vue";
import DynamicPriceIcon from "../components/MaterialIcon/DynamicPrice.vue";
import { type PriceSlot } from "../utils/forecast";
import formatter from "../mixins/formatter";
import store from "../store";

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
	name: "Energy",
	components: {
		TopHeader,
		IconSelectGroup,
		IconSelectItem,
		ForecastChart,
	},
	mixins: [formatter],
	data() {
		return {
			selectedType: TYPES.PRICE,
			types: Object.values(TYPES),
		};
	},
	head() {
		return { title: `${this.$t("sessions.title")} | evcc` };
	},
	computed: {
		typeIcons() {
			return {
				[TYPES.SOLAR]: "shopicon-regular-sun",
				[TYPES.PRICE]: DynamicPriceIcon,
				[TYPES.CO2]: "shopicon-regular-eco1",
			};
		},
		forecastAvailable() {
			const { grid, solar, co2 } = this.forecast;
			return grid || solar || co2;
		},
		forecast(): Forecast {
			return store.state?.forecast || {};
		},
		currency() {
			return store.state?.currency;
		},
	},
	methods: {
		updateType(type: string) {
			this.selectedType = type;
		},
	},
});
</script>

<style scoped></style>
