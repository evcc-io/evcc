<template>
	<div class="daybars">
		<div class="daybars-header d-flex justify-content-between align-items-center mb-2">
			<div class="daybars-title">{{ $t("main.pvTile.day24h") }}</div>
			<div class="daybars-total">{{ totalText }}</div>
		</div>
		<div class="daybars-grid" role="img" :aria-label="$t('main.pvTile.day24h')">
			<div
				v-for="(value, hour) in hourlyValues"
				:key="hour"
				class="daybar"
				:title="tooltip(hour, value)"
			>
				<div class="daybar-fill" :style="{ height: barHeight(value) }"></div>
			</div>
		</div>
		<div class="daybars-axis d-flex justify-content-between mt-1">
			<small>00</small>
			<small>06</small>
			<small>12</small>
			<small>18</small>
			<small>24</small>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import type { SolarDetails } from "../Forecast/types";

export default defineComponent({
	name: "PvDayBars",
	mixins: [formatter],
	props: {
		solar: { type: Object as PropType<SolarDetails> },
		pvEnergy: { type: Number, default: 0 },
		pvPower: { type: Number, default: 0 },
	},
	computed: {
		forecastHourlyValues(): number[] {
			const result = Array<number>(24).fill(0);
			const entries = this.solar?.timeseries || [];
			for (const entry of entries) {
				const ts = new Date(entry.ts);
				const now = new Date();
				if (ts.toDateString() !== now.toDateString()) continue;
				const hour = ts.getHours();
				result[hour] = Math.max(result[hour] || 0, entry.val || 0);
			}
			return result;
		},
		hasForecastValues(): boolean {
			return this.forecastHourlyValues.some((v) => v > 0);
		},
		syntheticHourlyValues(): number[] {
			const result = Array<number>(24).fill(0);
			const energyWh = Math.max(0, (this.pvEnergy || 0) * 1000);
			if (energyWh <= 0) {
				return result;
			}

			const sunrise = 6;
			const sunset = 20;
			const sunHours = sunset - sunrise;
			if (sunHours <= 0) {
				return result;
			}

			const weights = Array<number>(24).fill(0);
			let sumWeights = 0;
			for (let hour = sunrise; hour < sunset; hour++) {
				const x = (hour - sunrise + 0.5) / sunHours;
				const weight = Math.max(0, Math.sin(Math.PI * x));
				weights[hour] = weight;
				sumWeights += weight;
			}

			if (sumWeights <= 0) {
				return result;
			}

			const unit = energyWh / sumWeights;
			for (let hour = sunrise; hour < sunset; hour++) {
				result[hour] = Math.round(weights[hour] * unit);
			}

			const nowHour = new Date().getHours();
			if (nowHour >= 0 && nowHour < 24) {
				result[nowHour] = Math.max(result[nowHour], this.pvPower || 0);
			}

			return result;
		},
		hourlyValues(): number[] {
			return this.hasForecastValues ? this.forecastHourlyValues : this.syntheticHourlyValues;
		},
		maxValue(): number {
			return Math.max(...this.hourlyValues, 0);
		},
		totalEnergyWh(): number {
			if (!this.hasForecastValues && this.pvEnergy > 0) {
				return this.pvEnergy * 1000;
			}
			let sum = 0;
			for (const value of this.hourlyValues) {
				sum += value;
			}
			return sum;
		},
		totalText(): string {
			return this.fmtWh(this.totalEnergyWh, POWER_UNIT.AUTO);
		},
	},
	methods: {
		barHeight(value: number): string {
			if (this.maxValue <= 0) return "2px";
			const pct = Math.max(4, Math.round((value / this.maxValue) * 100));
			return `${pct}%`;
		},
		tooltip(hour: number, value: number): string {
			const from = String(hour).padStart(2, "0");
			const to = String((hour + 1) % 24).padStart(2, "0");
			return `${from}:00-${to}:00 ${this.fmtW(value, POWER_UNIT.AUTO)}`;
		},
	},
});
</script>

<style scoped>
.daybars {
	background: color-mix(in srgb, var(--evcc-box) 84%, var(--evcc-border));
	border-radius: 1rem;
	padding: 0.75rem;
}
.daybars-title {
	text-transform: uppercase;
	font-size: 12px;
	color: var(--evcc-gray);
}
.daybars-total {
	font-size: 12px;
	color: var(--evcc-gray);
}
.daybars-grid {
	height: 72px;
	display: grid;
	grid-template-columns: repeat(24, 1fr);
	align-items: end;
	column-gap: 2px;
}
.daybar {
	height: 100%;
	display: flex;
	align-items: end;
}
.daybar-fill {
	width: 100%;
	background: var(--evcc-yellow);
	border-radius: 2px 2px 0 0;
	opacity: 0.85;
}
.daybars-axis {
	color: var(--evcc-gray);
}
</style>
