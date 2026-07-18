<template>
	<div v-if="batteryAvailable" data-testid="battery-experimental">
		<BatteryStatusCards
			class="mb-4 box-pull-out"
			:battery="state.battery"
			:battery-mode="state.batteryMode"
		/>

		<BatteryHistoryCard
			class="mb-4 box-pull-out"
			:batteries="chartBatteries"
			:now="now"
			:kwh-available="kWhAvailable"
			@range-start="onRangeStart"
		/>

		<BatteryConfigCard
			class="mb-4 box-pull-out"
			:buffer-soc="state.bufferSoc"
			:priority-soc="state.prioritySoc"
			:buffer-start-soc="state.bufferStartSoc"
			:battery-discharge-control="state.batteryDischargeControl"
			:battery="state.battery"
		/>

		<Card
			v-if="gridChargeVisible"
			class="box-pull-out"
			:title="$t('batterySettings.gridChargeTab')"
		>
			<SmartCostLimit v-bind="smartCostLimitProps" />
		</Card>
	</div>
	<p v-else class="my-4 text-muted">{{ $t("batterySettings.noBattery") }}</p>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import store from "@/store";
import settings from "@/settings";
import api from "@/api";
import { SMART_COST_TYPE, CURRENCY, type BatteryMeter } from "@/types/evcc";
import Card from "../Helper/Card.vue";
import SmartCostLimit from "../Tariff/SmartCostLimit.vue";
import BatteryStatusCards from "./BatteryStatusCards.vue";
import BatteryConfigCard from "./BatteryConfigCard.vue";
import BatteryHistoryCard from "./BatteryHistoryCard.vue";
import { historyToSeries, forecastToSeries, buildChartBatteries } from "./history";
import type { BatteryHistorySeries, BatterySeries } from "./types";

const CHUNK_MS = 48 * 3600 * 1000; // 48h load/grow step

export default defineComponent({
	name: "BatteryExperimental",
	components: {
		Card,
		SmartCostLimit,
		BatteryStatusCards,
		BatteryConfigCard,
		BatteryHistoryCard,
	},
	data() {
		const now = new Date();
		return {
			now,
			rangeStart: new Date(now.getTime() - CHUNK_MS), // earliest time the chart currently shows
			loadedFrom: null as Date | null, // earliest time we have history for
			loadedTo: null as Date | null, // latest time we have history for
			loading: false,
			rawSeries: [] as BatteryHistorySeries[],
		};
	},
	computed: {
		state() {
			return store.state;
		},
		historyUpdated(): string | undefined {
			return store.state.historyUpdated;
		},
		devices(): BatteryMeter[] {
			return this.state.battery?.devices ?? [];
		},
		batteryAvailable(): boolean {
			return this.devices.length > 0;
		},
		evopt() {
			return this.state.evopt;
		},
		kWhAvailable(): boolean {
			return this.batteryAvailable && this.devices.every((d) => d.capacity > 0);
		},
		// earliest history we want loaded: a chunk before the visible start (kept as a buffer),
		// never later than what we already loaded so the window only grows
		requestedFrom(): Date {
			const wanted = new Date(this.rangeStart.getTime() - CHUNK_MS);
			return this.loadedFrom && this.loadedFrom < wanted ? this.loadedFrom : wanted;
		},
		chartBatteries(): BatterySeries[] {
			return buildChartBatteries(
				this.devices,
				historyToSeries(this.rawSeries),
				forecastToSeries(this.evopt, this.now.getTime())
			);
		},
		gridChargePossible(): boolean {
			return (
				this.devices.some(({ controllable }) => controllable) &&
				!!this.state.smartCostAvailable
			);
		},
		gridChargeLimit(): number | null {
			return this.state.batteryGridChargeLimit ?? null;
		},
		gridChargeVisible(): boolean {
			return this.gridChargePossible || this.gridChargeLimit !== null;
		},
		gridChargeTariff() {
			const { forecast, smartCostType } = this.state;
			return smartCostType === SMART_COST_TYPE.CO2 ? forecast?.co2 : forecast?.grid;
		},
		smartCostLimitProps() {
			return {
				currentLimit: this.gridChargeLimit,
				lastLimit: settings.lastBatterySmartCostLimit,
				smartCostType: this.state.smartCostType,
				currency: this.state.currency || CURRENCY.EUR,
				tariff: this.gridChargeTariff,
				possible: this.gridChargePossible,
			};
		},
	},
	watch: {
		// fetchHistory reloads only when the needed range is not already covered
		requestedFrom: {
			handler() {
				this.fetchHistory();
			},
			immediate: true,
		},
		now() {
			this.fetchHistory();
		},
		// bumped on each metrics persist; advance clock to reload recent history
		historyUpdated() {
			this.now = new Date();
		},
	},
	methods: {
		onRangeStart(start: Date) {
			this.rangeStart = start;
		},
		async fetchHistory() {
			if (this.loading || !this.batteryAvailable) return;
			const from = this.requestedFrom;
			const to = this.now;
			const backLoaded = this.loadedFrom !== null && this.loadedFrom <= from;
			const frontLoaded = this.loadedTo !== null && this.loadedTo >= to;
			if (backLoaded && frontLoaded) return;
			this.loading = true;
			try {
				const res = await api.get("history/energy", {
					params: {
						from: from.toISOString(),
						to: to.toISOString(),
						aggregate: "15m",
						grouped: false,
						group: "battery",
					},
				});
				this.rawSeries = res.data || [];
				this.loadedFrom = from;
				this.loadedTo = to;
			} catch (e) {
				console.error("failed to load battery history", e);
			} finally {
				this.loading = false;
			}
		},
	},
});
</script>
