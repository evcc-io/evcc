<template>
	<div v-if="isSolar && solarEnergy" class="row">
		<div class="col-6 col-sm-4 mb-3 d-flex flex-column">
			<div class="label">{{ label("today") }}</div>
			<div class="value d-flex flex-column flex-lg-row gap-lg-2 align-items-lg-baseline">
				<div class="text-primary text-nowrap">{{ solarEnergy.today.energy }}</div>
				<div class="extraValue text-nowrap">{{ label("remaining") }}</div>
			</div>
		</div>
		<div class="col-6 col-sm-4 mb-3 d-flex flex-column align-items-end align-items-sm-center">
			<div class="label">{{ label("tomorrow") }}</div>
			<div
				class="value d-flex flex-column flex-lg-row gap-lg-2 align-items-end align-items-sm-center align-items-lg-baseline"
			>
				<div class="text-primary text-nowrap">{{ solarEnergy.tomorrow.energy }}</div>
				<div v-if="solarEnergy.tomorrow.incomplete" class="extraValue text-nowrap">
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
					{{ solarEnergy.dayAfterTomorrow.energy }}
				</div>
				<div v-if="solarEnergy.dayAfterTomorrow.incomplete" class="extraValue text-nowrap">
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
import { defineComponent } from "vue";
import {
	energyByDay,
	dayStringByOffset,
	filterSlotsByDate,
	ForecastType,
	type PriceSlot,
} from "../utils/forecast";
import formatter, { POWER_UNIT } from "../mixins/formatter";

const LOCALES_WITHOUT_DAY_AFTER_TOMORROW = ["en", "tr"];

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
	mixins: [formatter],
	props: {
		type: { type: String as () => ForecastType, required: true },
		slots: { type: Array as () => PriceSlot[], default: () => [] },
		currency: { type: String },
	},
	computed: {
		isSolar() {
			return this.type === ForecastType.Solar;
		},
		upcomingSlots() {
			const now = new Date();
			return this.slots.filter((slot) => new Date(slot.end) > now).slice(0, 48);
		},
		solarEnergy(): EnergyByDay | undefined {
			if (!this.isSolar) return;

			const days = ["today", "tomorrow", "dayAfterTomorrow"];
			return days.reduce((acc, day, index) => {
				const energy = energyByDay(this.slots, index);
				const date = new Date();
				date.setDate(date.getDate() + index);
				const dayString = dayStringByOffset(index);
				const count = filterSlotsByDate(this.slots, dayString).length;
				const empty = count === 0;
				const complete = count === 24;
				const energyFmt = empty ? "-" : this.fmtWh(energy, POWER_UNIT.AUTO);
				acc[day] = {
					energy: energyFmt,
					incomplete: energy && !complete && !empty,
				};
				return acc;
			}, {} as EnergyByDay);
		},
		averagePrice() {
			if (this.isSolar) return "";
			const slots = this.upcomingSlots;
			const price = slots.reduce((acc, slot) => acc + slot.price, 0) / slots.length;
			return this.fmtValue(price, true);
		},
		priceRange() {
			if (this.isSolar) return "";
			const slots = this.upcomingSlots;
			const min = Math.min(...slots.map((slot) => slot.price));
			const max = Math.max(...slots.map((slot) => slot.price));
			return `${this.fmtValue(min, false)} – ${this.fmtValue(max, true)}`;
		},
		lowestPriceHour() {
			if (this.isSolar) return "";
			const slots = this.upcomingSlots;
			const min = Math.min(...slots.map((slot) => slot.price));
			const slot = slots.find((slot) => slot.price === min);
			if (!slot) return "";
			const start = new Date(slot.start);
			const end = new Date(slot.end);
			return `${this.weekdayShort(start)} ${this.hourShort(start)} – ${this.hourShort(end)}`;
		},
		highlightColor() {
			const colors = {
				[ForecastType.Price]: "text-price",
				[ForecastType.Co2]: "text-co2",
			};
			return colors[this.type];
		},
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
