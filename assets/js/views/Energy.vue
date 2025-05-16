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
import Header from "../components/Top/Header.vue";
import Chart from "../components/Forecast/Chart.vue";
import IconSelectItem from "../components/Helper/IconSelectItem.vue";
import IconSelectGroup from "../components/Helper/IconSelectGroup.vue";
import DynamicPriceIcon from "../components/MaterialIcon/DynamicPrice.vue";
import formatter from "../mixins/formatter.ts";
import store from "../store.ts";
import type { ForecastSlot, SolarDetails } from "../components/Forecast/types.ts";
import { ForecastType } from "../utils/forecast.ts";

interface Forecast {
	grid?: ForecastSlot[];
	solar?: SolarDetails;
	co2?: ForecastSlot[];
}

export default defineComponent({
	name: "Energy",
	components: {
		TopHeader: Header,
		IconSelectGroup,
		IconSelectItem,
		ForecastChart: Chart,
	},
	mixins: [formatter],
	data() {
		return {
			selectedType: ForecastType.Price,
			types: Object.values(ForecastType),
		};
	},
	head() {
		return { title: `${this.$t("sessions.title")} | evcc` };
	},
	computed: {
		typeIcons() {
			return {
				[ForecastType.Solar]: "shopicon-regular-sun",
				[ForecastType.Price]: DynamicPriceIcon,
				[ForecastType.Co2]: "shopicon-regular-eco1",
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
		updateType(type: ForecastType) {
			this.selectedType = type;
		},
	},
});
</script>

<style scoped></style>
