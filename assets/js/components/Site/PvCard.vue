<template>
	<div
		class="pv-card d-flex flex-column pt-4 pb-2 px-3 px-sm-4 mx-2 mx-sm-0"
		:class="{ 'pv-active': pvPower > 0 }"
	>
		<div class="d-flex justify-content-between align-items-center mb-3 text-truncate">
			<h3 class="me-2 mb-0 text-truncate d-flex align-items-center">
				<shopicon-regular-sun class="me-2 flex-shrink-0" :class="{ 'sun-active': pvPower > 0 }" />
				<span class="text-truncate">{{ cardTitle }}</span>
			</h3>
			<button type="button" class="btn btn-sm btn-outline-secondary" @click.stop="openHistory">
				{{ $t("main.pvTile.history") }}
			</button>
		</div>

		<div class="pv-hero mb-3">
			<div class="pv-hero-label">{{ $t("main.pvTile.currentPower") }}</div>
			<div class="pv-hero-value">{{ fmtPower(pvPower) }}</div>
		</div>

		<div class="details d-flex align-items-start mb-2">
			<LabelAndValue
				:label="$t('main.pvTile.generatedToday')"
					:value="displayGeneratedTodayEnergy"
				:valueFmt="fmtEnergyKWh"
				align="center"
			/>
			<LabelAndValue
				:label="$t('main.pvTile.forecastPossible')"
				:value="forecastRemainingToday"
				:valueFmt="fmtEnergyWh"
				align="end"
			/>
		</div>
		<hr class="divider" />

		<div class="flex-grow-1 d-flex flex-column justify-content-end pt-3 pb-2 gap-3">
			<PvDayBars
				:solar="forecast?.solar"
				:pvEnergy="displayGeneratedTodayEnergy"
				:pvPower="pvPower"
			/>

			<div>
				<div class="summary-label">{{ $t("main.pvTile.theoreticalTotal") }}</div>
				<h3 class="summary-value m-0">
					{{ theoreticalTotalText }}
				</h3>
			</div>

			<div v-if="pv.length > 0" class="systems d-flex flex-column gap-2">
				<div class="summary-label">{{ $t("main.pvTile.systems") }}</div>
				<div
					v-for="(plant, index) in pv"
					:key="index"
					class="system-row d-flex justify-content-between align-items-center gap-3"
				>
					<span class="text-truncate">{{ plant.title || genericPvTitle(index) }}</span>
					<span class="text-nowrap fw-bold">{{ fmtPower(plant.power) }}</span>
				</div>
			</div>
		</div>

		<PvHistoryModal :pv="pv" :pvEnergy="displayGeneratedTodayEnergy" :forecast="forecast" />
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/sun";
import Modal from "bootstrap/js/dist/modal";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import LabelAndValue from "../Helper/LabelAndValue.vue";
import PvDayBars from "./PvDayBars.vue";
import PvHistoryModal from "./PvHistoryModal.vue";
import { defineComponent, type PropType } from "vue";
import type { Forecast, Meter } from "@/types/evcc";
import settings from "@/settings";

export default defineComponent({
	name: "PvCard",
	components: { LabelAndValue, PvDayBars, PvHistoryModal },
	mixins: [formatter],
	props: {
		pv: { type: Array as PropType<Meter[]>, default: () => [] },
		pvPower: { type: Number, default: 0 },
		pvEnergy: Number,
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
		experimental: Boolean,
	},
	computed: {
		generatedTodayEnergy() {
			const energyCandidates: number[] = [];
			if (typeof this.pvEnergy === "number") {
				energyCandidates.push(this.pvEnergy);
			}
			const firstMeterEnergy = this.pv?.[0]?.energy;
			if (typeof firstMeterEnergy === "number") {
				energyCandidates.push(firstMeterEnergy);
			}
			if (energyCandidates.length > 0) {
				return Math.max(...energyCandidates);
			}
			return 0;
		},
		displayGeneratedTodayEnergy() {
			if (this.generatedTodayEnergy > 0) {
				return this.generatedTodayEnergy;
			}
			// Fallback for demo/live startup phases where energy counters lag behind power.
			if (this.pvPower > 0) {
				const now = new Date();
				const hour = now.getHours() + now.getMinutes() / 60;
				const effectiveSunHours = Math.max(0, Math.min(14, hour - 6));
				return (this.pvPower / 1000) * effectiveSunHours * 0.5;
			}
			return 0;
		},
		cardTitle() {
			if (this.pv.length === 1 && this.pv[0]?.title) {
				return this.pv[0].title;
			}
			return this.$t("main.energyflow.pv");
		},
		forecastRemainingToday() {
			if (!this.forecast?.solar) {
				return undefined;
			}
			const { today, scale } = this.forecast.solar;
			const factor = this.experimental && settings.solarAdjusted && scale ? scale : 1;
			const energy = today?.energy || 0;
			return energy * factor;
		},
		theoreticalTotal() {
			if (
				typeof this.generatedTodayEnergy !== "number" &&
				typeof this.forecastRemainingToday !== "number"
			) {
				return undefined;
			}
			// generatedTodayEnergy is kWh, forecastRemainingToday is Wh
			return (this.displayGeneratedTodayEnergy || 0) * 1000 + (this.forecastRemainingToday || 0);
		},
		theoreticalTotalText() {
			if (typeof this.theoreticalTotal !== "number") {
				return "-";
			}
			return this.fmtEnergyWh(this.theoreticalTotal);
		},
	},
	methods: {
		openHistory() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("pvHistoryModal") as HTMLElement
			);
			modal.show();
		},
		fmtPower(value: number) {
			return this.fmtW(value, POWER_UNIT.AUTO);
		},
		fmtEnergyWh(value: number) {
			// forecast and most chart energy values are in Wh
			return this.fmtWh(value, POWER_UNIT.KW);
		},
		fmtEnergyKWh(value: number) {
			// pvEnergy from state is in kWh
			return this.fmtWh(value * 1000, POWER_UNIT.KW);
		},
		genericPvTitle(index: number) {
			return `${this.$t("config.devices.solarSystem")} #${index + 1}`;
		},
	},
});
</script>

<style scoped>
.pv-card {
	border-radius: 2rem;
	color: var(--evcc-default-text);
	background: var(--evcc-box);
	position: relative;
	overflow: hidden;
	transition: box-shadow 0.4s ease;
}

.pv-active {
	box-shadow: 0 0 0 2px var(--evcc-yellow), 0 4px 24px 0 color-mix(in srgb, var(--evcc-yellow) 30%, transparent);
}

.pv-hero-label {
	text-transform: uppercase;
	color: var(--evcc-gray);
	font-size: 14px;
	line-height: 1.1;
	margin-bottom: 0.25rem;
}

.pv-hero-value {
	font-size: 44px;
	font-weight: 700;
	line-height: 1;
	letter-spacing: -0.02em;
}

@keyframes spin-slow {
	from { transform: rotate(0deg); }
	to   { transform: rotate(360deg); }
}

.sun-active {
	animation: spin-slow 8s linear infinite;
	color: var(--evcc-yellow);
}

.details > * {
	flex-grow: 1;
	flex-shrink: 1;
	flex-basis: 0;
	min-width: 0;
}

.details :deep(.label) {
	white-space: normal;
	line-height: 1.2;
	overflow-wrap: anywhere;
	text-wrap: balance;
}

.divider {
	border: none;
	border-bottom-width: 1px;
	border-bottom-style: solid;
	border-bottom-color: var(--evcc-gray);
	background: none;
	opacity: 0.5;
	margin: 0 -1rem;
}

.summary-label {
	text-transform: uppercase;
	color: var(--evcc-gray);
	font-size: 14px;
	margin-bottom: 0.25rem;
}

.summary-value {
	font-size: 32px;
}

.system-row {
	font-size: 18px;
	min-width: 0;
}

@media (min-width: 576px) {
	.divider {
		margin: 0 -1.5rem;
	}
}

@media (max-width: 767.98px) {
	.pv-hero-value {
		font-size: 36px;
	}
	.details {
		flex-wrap: wrap;
		row-gap: 0.25rem;
	}
	.details > * {
		flex: 1 1 calc(50% - 0.5rem);
	}
}
</style>