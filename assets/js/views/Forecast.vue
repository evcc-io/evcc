<template>
	<div
		class="container px-4 safe-area-inset d-flex flex-column"
		:class="{ 'empty-container': !forecastAvailable }"
	>
		<TopHeader :title="$t('forecast.modalTitle')" />
		<div v-if="!forecastAvailable" class="flex-grow-1 d-flex">
			<div class="empty-box d-flex flex-column p-5">
				<ul class="list-unstyled mb-4">
					<li class="d-flex align-items-start gap-2 mb-3">
						<shopicon-regular-sun
							size="s"
							class="text-muted flex-shrink-0"
						></shopicon-regular-sun>
						<div>
							<strong>{{ $t("forecast.type.solar") }}</strong>
							<div class="text-muted">{{ $t("forecast.empty.solar") }}</div>
						</div>
					</li>
					<li class="d-flex align-items-start gap-2 mb-3">
						<DynamicPriceIcon class="text-muted flex-shrink-0" />
						<div>
							<strong>{{ $t("forecast.type.price") }}</strong>
							<div class="text-muted">{{ $t("forecast.empty.price") }}</div>
						</div>
					</li>
					<li class="d-flex align-items-start gap-2 mb-3">
						<shopicon-regular-eco1
							size="s"
							class="text-muted flex-shrink-0"
						></shopicon-regular-eco1>
						<div>
							<strong>{{ $t("forecast.type.co2") }}</strong>
							<div class="text-muted">{{ $t("forecast.empty.co2") }}</div>
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
				<section v-if="forecast.solar" class="mb-5">
					<div class="d-flex align-items-baseline my-4">
						<h3 class="fw-normal mb-0">{{ $t("forecast.type.solar") }}</h3>
						<div
							v-if="showSolarAdjust"
							class="form-check form-switch ms-auto mb-0 text-nowrap"
						>
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
					</div>
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
				</section>

				<section
					v-if="forecast.grid"
					class="mb-5"
					:style="isGridStatic ? { order: 1 } : undefined"
				>
					<div class="d-flex align-items-baseline my-4">
						<h3 class="fw-normal mb-0">{{ $t("forecast.type.price") }}</h3>
						<div class="form-check form-switch ms-auto mb-0 text-nowrap">
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
					</div>
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
				</section>

				<section v-if="forecast.co2" class="mb-5">
					<h3 class="fw-normal my-4">{{ $t("forecast.type.co2") }}</h3>
					<div class="chart-edge">
						<Co2Chart
							:co2="forecast.co2"
							:chart-width="chartWidth"
							:end-date="chartEndDate"
							:scroll-left="scrollLeft"
							@scroll="onChartScroll"
						/>
					</div>
					<Co2Details :co2="forecast.co2" />
				</section>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/eco1";
import { defineComponent } from "vue";
import Header from "../components/Top/Header.vue";
import DynamicPriceIcon from "../components/MaterialIcon/DynamicPrice.vue";
import SolarChart from "../components/Forecast/SolarChart.vue";
import SolarDetails from "../components/Forecast/SolarDetails.vue";
import PriceChart from "../components/Forecast/PriceChart.vue";
import GridDetails from "../components/Forecast/GridDetails.vue";
import Co2Chart from "../components/Forecast/Co2Chart.vue";
import Co2Details from "../components/Forecast/Co2Details.vue";
import formatter from "@/mixins/formatter";
import settings from "@/settings";
import store from "../store";
import { adjustedSolar, ForecastType, isStaticTariff } from "@/utils/forecast";

const MIN_HOURS = 76;
const MAX_HOURS = 96;

export default defineComponent({
	name: "Forecast",
	components: {
		TopHeader: Header,
		DynamicPriceIcon,
		SolarChart,
		SolarDetails,
		PriceChart,
		GridDetails,
		Co2Chart,
		Co2Details,
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
			return !!(this.forecast.grid || this.forecast.solar || this.forecast.co2);
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
			for (const s of this.forecast.grid || []) {
				const t = new Date(s.end).getTime();
				if (t > latest) latest = t;
			}
			for (const s of this.forecast.co2 || []) {
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
			return settings.solarAdjusted;
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
			// TODO fix https://github.com/evcc-io/evcc/issues/29165
			return !!this.forecast.solar && this.experimental && false;
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
		changeAdjusted() {
			settings.solarAdjusted = !settings.solarAdjusted;
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
	.chart-edge {
		margin-left: -1.5rem;
		margin-right: -1.5rem;
	}
}
</style>
