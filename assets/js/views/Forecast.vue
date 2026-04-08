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
			<main class="col-12">
				<section v-if="forecast.solar" class="mb-5">
					<div class="d-flex flex-wrap gap-3 align-items-baseline my-4">
						<h3
							class="fw-normal mb-0 d-flex gap-3 flex-wrap align-items-baseline overflow-hidden"
						>
							<span class="d-block text-nowrap text-truncate">{{
								$t("forecast.type.solar")
							}}</span>
							<small v-if="solarSubtitle" class="d-block text-nowrap text-truncate">{{
								solarSubtitle
							}}</small>
						</h3>
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
								{{ solarAdjustText }}
							</label>
						</div>
					</div>
					<SolarChart
						:solar="solar"
						:raw-solar="forecast.solar"
						:chart-width="chartWidth"
						:end-date="chartEndDate"
						:scroll-left="scrollLeft"
						@scroll="onChartScroll"
					/>
				</section>

				<section v-if="forecast.grid" class="mb-5">
					<div class="d-flex flex-wrap gap-3 align-items-baseline my-4">
						<h3
							class="fw-normal mb-0 d-flex gap-3 flex-wrap align-items-baseline overflow-hidden"
						>
							<span class="d-block text-nowrap text-truncate">{{
								$t("forecast.type.price")
							}}</span>
							<small v-if="priceSubtitle" class="d-block text-nowrap text-truncate">{{
								priceSubtitle
							}}</small>
						</h3>
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
					<PriceChart
						:grid="forecast.grid"
						:currency="currency"
						:zoom="priceZoom"
						:chart-width="chartWidth"
						:end-date="chartEndDate"
						:scroll-left="scrollLeft"
						@scroll="onChartScroll"
					/>
				</section>

				<section v-if="forecast.co2">
					<h3
						class="fw-normal my-4 d-flex gap-3 flex-wrap align-items-baseline overflow-hidden"
					>
						<span class="d-block text-nowrap text-truncate">{{
							$t("forecast.type.co2")
						}}</span>
						<small v-if="co2Subtitle" class="d-block text-nowrap text-truncate">{{
							co2Subtitle
						}}</small>
					</h3>
					<Co2Chart
						:co2="forecast.co2"
						:chart-width="chartWidth"
						:end-date="chartEndDate"
						:scroll-left="scrollLeft"
						@scroll="onChartScroll"
					/>
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
import PriceChart from "../components/Forecast/PriceChart.vue";
import Co2Chart from "../components/Forecast/Co2Chart.vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import settings from "@/settings";
import store from "../store";
import { adjustedSolar, ForecastType } from "@/utils/forecast";
import type { ForecastSlot } from "../components/Forecast/types";

const MIN_HOURS = 76;
const MAX_HOURS = 96;
const SLOTS_PER_HOUR = 4;

export default defineComponent({
	name: "Forecast",
	components: {
		TopHeader: Header,
		DynamicPriceIcon,
		SolarChart,
		PriceChart,
		Co2Chart,
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
		showSolarAdjust() {
			return !!this.forecast.solar && this.experimental;
		},
		solar() {
			return this.showSolarAdjust && this.solarAdjusted
				? adjustedSolar(this.forecast.solar)
				: this.forecast.solar;
		},
		solarAdjustText() {
			const text = this.$t("forecast.solarAdjustShort");
			const scale = this.forecast.solar?.scale || 1;
			const percentDiff = scale * 100 - 100;
			return `${text} (${this.fmtPercentage(percentDiff, 0, true)})`;
		},
		solarSubtitle(): string {
			const s = this.solar;
			if (!s) return "";
			const today = s.today?.energy ? this.fmtWh(s.today.energy, POWER_UNIT.AUTO) : "";
			const tomorrow = s.tomorrow?.energy
				? this.fmtWh(s.tomorrow.energy, POWER_UNIT.AUTO)
				: "";
			if (!today && !tomorrow) return "";
			const parts = [];
			if (today) parts.push(`${today} ${this.$t("forecast.solar.remaining")}`);
			if (tomorrow) parts.push(`${tomorrow} ${this.$t("forecast.solar.tomorrow")}`);
			return parts.join(" ・ ");
		},
		priceSubtitle(): string {
			const slots = this.upcomingSlots(this.forecast.grid);
			if (slots.length === 0) return "";
			const values = slots.map((s) => s.value);
			const min = Math.min(...values);
			const max = Math.max(...values);
			const avg = values.reduce((a, b) => a + b, 0) / values.length;
			const fmtMin = this.fmtPricePerKWh(min, this.currency, false, false);
			const fmtMax = this.fmtPricePerKWh(max, this.currency, false, true);
			const fmtAvg = this.fmtPricePerKWh(avg, this.currency, false, true);
			return `⌀ ${fmtAvg} ・ ${fmtMin} – ${fmtMax}`;
		},
		co2Subtitle(): string {
			const slots = this.upcomingSlots(this.forecast.co2);
			if (slots.length === 0) return "";
			const values = slots.map((s) => s.value);
			const min = Math.min(...values);
			const max = Math.max(...values);
			const avg = values.reduce((a, b) => a + b, 0) / values.length;
			const fmtMin = this.fmtNumber(min, 0);
			const fmtMax = this.fmtCo2Medium(max);
			const fmtAvg = this.fmtCo2Medium(avg);
			return `⌀ ${fmtAvg} ・ ${fmtMin} – ${fmtMax}`;
		},
	},
	methods: {
		changeAdjusted() {
			settings.solarAdjusted = !settings.solarAdjusted;
		},
		togglePriceZoom() {
			settings.priceZoom = !settings.priceZoom;
		},
		onChartScroll(val: number) {
			if (this.isScrolling) return;
			this.isScrolling = true;
			this.scrollLeft = val;
			this.$nextTick(() => {
				this.isScrolling = false;
			});
		},
		upcomingSlots(slots?: ForecastSlot[]): ForecastSlot[] {
			if (!Array.isArray(slots)) return [];
			const now = new Date();
			return slots
				.filter((slot) => new Date(slot.end) > now)
				.slice(0, MAX_HOURS * SLOTS_PER_HOUR);
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
</style>
