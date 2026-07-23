<template>
	<div
		class="container px-4 safe-area-inset d-flex flex-column"
		:class="{ 'empty-container': !forecastAvailable }"
	>
		<TopHeader :title="$t('forecast.modalTitle')" />
		<div v-if="!forecastAvailable" class="flex-grow-1 d-flex">
			<div class="empty-box d-flex flex-column p-5">
				<ul class="list-unstyled mb-4">
					<li
						v-for="{ type, icon } in emptyItems"
						:key="type"
						class="d-flex align-items-start gap-2 mb-3"
					>
						<component :is="icon" size="s" class="text-muted flex-shrink-0" />
						<div>
							<strong>{{ $t(`forecast.type.${type}`) }}</strong>
							<div class="text-muted">{{ $t(`forecast.empty.${type}`) }}</div>
						</div>
					</li>
				</ul>
				<router-link to="/config#tariffs" class="btn btn-outline-primary">
					{{ $t("forecast.empty.setup") }}
				</router-link>
			</div>
		</div>
		<div v-else class="row">
			<main class="col-12 d-flex flex-column">
				<Card
					v-if="forecast.solar"
					:title="$t('forecast.type.solar')"
					edge-to-edge
					class="box-pull-out mb-4"
				>
					<template v-if="showSolarAdjust" #actions>
						<div class="form-check form-switch mb-0 text-nowrap">
							<input
								id="solarForecastAdjust"
								:checked="solarAdjusted"
								class="form-check-input"
								type="checkbox"
								role="switch"
								@change="changeAdjusted"
							/>
							<label class="form-check-label text-muted" for="solarForecastAdjust">
								<span class="d-md-none">{{ solarAdjustTextShort }}</span>
								<span class="d-none d-md-inline">{{ solarAdjustTextMedium }}</span>
							</label>
						</div>
					</template>
					<div class="chart-edge">
						<SolarChart
							:solar="solar"
							:raw-solar="forecast.solar"
							:chart-width="chartWidth"
							:end-date="chartEndDate"
							:scroll-left="scrollLeft"
							@scroll="onChartScroll"
						/>
					</div>
					<SolarDetails :solar="solar" />
				</Card>

				<Card
					v-if="forecast.grid"
					:title="$t('forecast.type.price')"
					edge-to-edge
					class="box-pull-out mb-4"
					:style="isGridStatic ? { order: 1 } : undefined"
				>
					<template #actions>
						<div class="form-check form-switch mb-0 text-nowrap">
							<input
								id="priceZoom"
								:checked="priceZoom"
								class="form-check-input"
								type="checkbox"
								role="switch"
								@change="togglePriceZoom"
							/>
							<label class="form-check-label text-muted" for="priceZoom">
								{{ $t("forecast.priceZoom") }}
							</label>
						</div>
					</template>
					<div class="chart-edge">
						<PriceChart
							:grid="forecast.grid"
							:feedin="showFeedin ? forecast.feedin : undefined"
							:currency="currency"
							:zoom="priceZoom"
							:chart-width="chartWidth"
							:end-date="chartEndDate"
							:scroll-left="scrollLeft"
							@scroll="onChartScroll"
						/>
					</div>
					<GridDetails
						:grid="forecast.grid"
						:feedin="forecast.feedin"
						:currency="currency"
						:show-feedin="showFeedin"
						@toggle-feedin="toggleFeedin"
					/>
				</Card>

				<Card
					v-for="t in valueForecastTypes"
					:key="t"
					:title="$t(`forecast.type.${t}`)"
					edge-to-edge
					class="box-pull-out mb-4"
				>
					<div class="chart-edge">
						<ValueChart
							:type="t"
							:rates="forecast[t]!"
							:chart-width="chartWidth"
							:end-date="chartEndDate"
							:scroll-left="scrollLeft"
							@scroll="onChartScroll"
						/>
					</div>
					<ValueDetails :type="t" :rates="forecast[t]" />
				</Card>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/thermometerhalf";
import { defineComponent, markRaw } from "vue";
import Header from "../components/Top/Header.vue";
import Card from "../components/Helper/Card.vue";
import DynamicPriceIcon from "../components/MaterialIcon/DynamicPrice.vue";
import SolarChart from "../components/Forecast/SolarChart.vue";
import SolarDetails from "../components/Forecast/SolarDetails.vue";
import PriceChart from "../components/Forecast/PriceChart.vue";
import GridDetails from "../components/Forecast/GridDetails.vue";
import ValueChart, { type ValueChartType } from "../components/Forecast/ValueChart.vue";
import ValueDetails from "../components/Forecast/ValueDetails.vue";
import formatter from "@/mixins/formatter";
import api from "@/api";
import settings from "@/settings";
import store from "../store";
import { adjustedSolar, ForecastType, isStaticTariff } from "@/utils/forecast";

const MIN_HOURS = 76;
const MAX_HOURS = 96;

export default defineComponent({
	name: "Forecast",
	components: {
		TopHeader: Header,
		Card,
		DynamicPriceIcon,
		SolarChart,
		SolarDetails,
		PriceChart,
		GridDetails,
		ValueChart,
		ValueDetails,
	},
	mixins: [formatter],
	data() {
		return { ForecastType, scrollLeft: 0, isScrolling: false };
	},
	head() {
		return { title: this.$t("forecast.modalTitle") };
	},
	computed: {
		forecast() {
			return store.state?.forecast || {};
		},
		forecastAvailable() {
			return !!(
				this.forecast.grid ||
				this.forecast.solar ||
				this.forecast.co2 ||
				this.forecast.temperature
			);
		},
		valueForecastTypes(): ValueChartType[] {
			return (["co2", "temperature"] as ValueChartType[]).filter((t) => this.forecast[t]);
		},
		emptyItems() {
			return [
				{ type: "solar", icon: "shopicon-regular-sun" },
				{ type: "price", icon: markRaw(DynamicPriceIcon) },
				{ type: "co2", icon: "shopicon-regular-eco1" },
				{ type: "temperature", icon: "shopicon-regular-thermometerhalf" },
			];
		},
		startDate(): Date {
			const now = new Date();
			now.setMinutes(0, 0, 0);
			return now;
		},
		maxEndDate(): Date {
			const end = new Date(this.startDate);
			end.setHours(end.getHours() + MAX_HOURS);
			return end;
		},
		dataEndDate(): Date {
			let latest = this.startDate.getTime();
			for (const s of [
				...(this.forecast.grid || []),
				...(this.forecast.co2 || []),
				...(this.forecast.temperature || []),
			]) {
				const t = new Date(s.end).getTime();
				if (t > latest) latest = t;
			}
			for (const e of this.forecast.solar?.timeseries || []) {
				const t = new Date(e.ts).getTime();
				if (t > latest) latest = t;
			}
			const end = new Date(Math.min(latest, this.maxEndDate.getTime()));
			return end;
		},
		chartEndDate(): Date {
			const minEnd = new Date(this.startDate);
			minEnd.setHours(minEnd.getHours() + MIN_HOURS);
			return this.dataEndDate > minEnd ? this.dataEndDate : minEnd;
		},
		chartWidth(): number {
			const ms = this.chartEndDate.getTime() - this.startDate.getTime();
			const slots = Math.ceil(ms / (15 * 60 * 1000));
			return slots * 4 + 56;
		},
		currency() {
			return store.state?.currency;
		},
		experimental() {
			return store.state?.experimental;
		},
		solarAdjusted() {
			return store.state?.solarAdjusted;
		},
		priceZoom() {
			return settings.priceZoom;
		},
		showFeedin() {
			return !settings.hideFeedin;
		},
		isGridStatic(): boolean {
			if (!this.forecast.grid) return false;
			if (!isStaticTariff(this.forecast.grid)) return false;
			if (this.forecast.feedin?.length) {
				return isStaticTariff(this.forecast.feedin);
			}
			return true;
		},
		showSolarAdjust() {
			return !!this.forecast.solar && this.experimental;
		},
		solar() {
			return this.showSolarAdjust && this.solarAdjusted
				? adjustedSolar(this.forecast.solar)
				: this.forecast.solar;
		},
		solarAdjustPercent(): string {
			const scale = this.forecast.solar?.scale || 1;
			const percentDiff = scale * 100 - 100;
			return this.fmtPercentage(percentDiff, 0, true);
		},
		solarAdjustTextShort(): string {
			return `${this.$t("forecast.solarAdjustShort")} (${this.solarAdjustPercent})`;
		},
		solarAdjustTextMedium(): string {
			return `${this.$t("forecast.solarAdjustMedium")} (${this.solarAdjustPercent})`;
		},
	},
	methods: {
		async changeAdjusted(e: Event) {
			try {
				await api.post(
					`solaradjusted/${(e.target as HTMLInputElement).checked ? "true" : "false"}`
				);
			} catch (err) {
				console.error(err);
			}
		},
		togglePriceZoom() {
			settings.priceZoom = !settings.priceZoom;
		},
		toggleFeedin() {
			settings.hideFeedin = !settings.hideFeedin;
		},
		onChartScroll(val: number) {
			if (this.isScrolling) return;
			this.isScrolling = true;
			this.scrollLeft = val;
			this.$nextTick(() => {
				this.isScrolling = false;
			});
		},
	},
});
</script>

<style scoped>
.empty-container {
	min-height: calc(100dvh - var(--bottom-space));
}
.empty-box {
	background-color: var(--evcc-box);
	margin: auto;
	border-radius: 2rem;
	max-width: 480px;
}
@media (max-width: 575.98px) {
	/* cancel the card's padding for full-bleed charts */
	.chart-edge {
		margin-left: -1.5rem;
		margin-right: -1.5rem;
	}
}
@media (min-width: 576px) {
	/* compensate the card border so the chart width fits without scrolling on wide screens */
	.chart-edge {
		margin-left: -1px;
		margin-right: -1px;
	}
}
</style>
