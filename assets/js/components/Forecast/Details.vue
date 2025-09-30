<template>
	<div v-if="isSolar && solar" class="row">
		<div class="col-6 col-sm-4 mb-3 d-flex flex-column">
			<div class="label">{{ label("today") }}</div>
			<div class="value d-flex flex-column flex-lg-row gap-lg-2 align-items-lg-baseline">
				<div class="text-primary text-nowrap">
					<AnimatedNumber :to="solar.today?.energy" :format="fmtEnergy" />
				</div>
				<div class="extraValue text-nowrap">{{ label("remaining") }}</div>
			</div>
		</div>
		<div class="col-6 col-sm-4 mb-3 d-flex flex-column align-items-end align-items-sm-center">
			<div class="label">{{ label("tomorrow") }}</div>
			<div
				class="value d-flex flex-column flex-lg-row gap-lg-2 align-items-end align-items-sm-center align-items-lg-baseline"
			>
				<div class="text-primary text-nowrap">
					<AnimatedNumber :to="solar.tomorrow?.energy" :format="fmtEnergy" />
				</div>
				<div v-if="!solar.tomorrow?.complete" class="extraValue text-nowrap">
					{{ label("partly") }}
				</div>
			</div>
		</div>
		<div class="col-6 col-sm-4 mb-3 d-flex flex-column align-items-start align-items-sm-end">
			<div class="label">{{ label("dayAfterTomorrow") }}</div>
			<div
				class="value d-flex flex-column flex-lg-row gap-lg-2 align-items-start align-items-sm-end align-items-lg-baseline"
			>
				<div class="text-primary text-nowrap">
					<AnimatedNumber :to="solar.dayAfterTomorrow?.energy" :format="fmtEnergy" />
				</div>
				<div v-if="!solar.dayAfterTomorrow?.complete" class="extraValue text-nowrap">
					{{ label("partly") }}
				</div>
			</div>
		</div>
	</div>
	<div v-else class="row">
		<div class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column">
			<div class="label">{{ label("range") }}</div>
			<div class="value text-price text-nowrap" :class="highlightColor">
				{{ priceRange }}
			</div>
		</div>
		<div
			class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column align-items-sm-end align-items-lg-center"
		>
			<div class="label">{{ label("average") }}</div>
			<div class="value text-price text-nowrap" :class="highlightColor">
				{{ averagePrice }}
			</div>
		</div>
		<div
			class="col-12 col-sm-6 col-lg-4 mb-3 d-flex flex-column align-items-sm-start align-items-lg-end"
		>
			<div class="label">{{ label("lowestHour") }}</div>
			<div class="value text-price text-nowrap" :class="highlightColor">
				{{ lowestPriceHour }}
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import type { CURRENCY, Timeout } from "@/types/evcc";
import { ForecastType, findLowestSumSlotIndex } from "@/utils/forecast";
import type { ForecastSlot, SolarDetails } from "./types";
const LOCALES_WITHOUT_DAY_AFTER_TOMORROW = ["en", "tr"];

const FORECASTED_HOURS = 48;
const SLOTS_PER_HOUR = 4;

export interface Energy {
	energy: string;
	incomplete: boolean;
}

export interface EnergyByDay {
	today: Energy;
	tomorrow: Energy;
	dayAfterTomorrow: Energy;
}

export default defineComponent({
	name: "ForecastDetails",
	components: {
		AnimatedNumber,
	},
	mixins: [formatter],
	props: {
		type: { type: String as () => ForecastType, required: true },
		grid: { type: Array as PropType<ForecastSlot[]> },
		co2: { type: Array as PropType<ForecastSlot[]> },
		solar: { type: Object as PropType<SolarDetails> },
		currency: { type: String as PropType<CURRENCY> },
	},
	data() {
		return {
			now: new Date(),
			interval: null as Timeout,
		};
	},
	computed: {
		isSolar() {
			return this.type === ForecastType.Solar;
		},
		isPrice() {
			return this.type === ForecastType.Price;
		},
		upcomingSlots(): ForecastSlot[] {
			const now = this.now;
			const slots = this.isPrice ? this.grid || [] : this.co2 || [];
			return slots
				.filter((slot) => new Date(slot.end) > now)
				.slice(0, FORECASTED_HOURS * SLOTS_PER_HOUR);
		},
		averagePrice() {
			if (this.isSolar) return "";
			const slots = this.upcomingSlots;
			const price = slots.reduce((acc, slot) => acc + slot.value, 0) / slots.length;
			return this.fmtValue(price, true);
		},
		priceRange() {
			if (this.isSolar) return "";
			const slots = this.upcomingSlots;
			const min = Math.min(...slots.map((slot) => slot.value));
			const max = Math.max(...slots.map((slot) => slot.value));
			return `${this.fmtValue(min, false)} – ${this.fmtValue(max, true)}`;
		},
		lowestPriceHour() {
			if (this.isSolar) return "";
			const slots = this.upcomingSlots;
			const index = findLowestSumSlotIndex(slots, SLOTS_PER_HOUR);
			if (index === -1) return "";
			const startSlot = slots[index];
			const endSlot = slots[index + SLOTS_PER_HOUR - 1];
			if (!startSlot || !endSlot) return "";
			const start = new Date(startSlot.start);
			const end = new Date(endSlot.end);

			return `${this.weekdayShort(start)} ${this.fmtHourMinute(start)} – ${this.fmtHourMinute(end)}`;
		},
		highlightColor() {
			switch (this.type) {
				case ForecastType.Price:
					return "text-price";
				case ForecastType.Co2:
					return "text-co2";
				default:
					return "";
			}
		},
	},
	mounted() {
		this.now = new Date();
		this.interval = setInterval(() => {
			this.now = new Date();
		}, 1000 * 60);
	},
	beforeUnmount() {
		if (this.interval) {
			clearInterval(this.interval);
		}
	},
	methods: {
		label(key: string) {
			// special case "day after tomorrow"
			if (
				key === "dayAfterTomorrow" &&
				LOCALES_WITHOUT_DAY_AFTER_TOMORROW.includes(this.$i18n.locale)
			) {
				const date = new Date();
				date.setDate(date.getDate() + 2);
				return this.fmtDayMonth(date);
			}

			return this.$t(`forecast.${this.type}.${key}`);
		},
		fmtValue(value: number, withUnit = true) {
			if (this.type === ForecastType.Price) {
				return this.fmtPricePerKWh(value, this.currency, false, withUnit);
			}
			return withUnit ? this.fmtCo2Medium(value) : this.fmtNumber(value, 0);
		},
		fmtEnergy(energy: number | undefined) {
			return !energy ? "-" : this.fmtWh(energy, POWER_UNIT.AUTO);
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
