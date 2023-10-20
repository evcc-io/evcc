<template>
	<div class="plan my-3">
		<div class="justify-content-between mb-2 d-flex justify-content-between">
			<div class="text-start">
				<div class="label">{{ $t("main.targetChargePlan.chargeDuration") }}</div>
				<div class="value text-primary">{{ planDuration }}</div>
			</div>
			<div v-if="hasTariff" class="text-end">
				<div class="label">
					<span v-if="activeSlot">{{ activeSlotName }}</span>
					<span v-else-if="isCo2">{{ $t("main.targetChargePlan.co2Label") }}</span>
					<span v-else>{{ $t("main.targetChargePlan.priceLabel") }}</span>
				</div>
				<div class="value text-primary">
					{{ fmtAvgPrice }}
				</div>
			</div>
		</div>
		<TariffChart :slots="slots" @slot-hovered="slotHovered" />
	</div>
</template>

<script>
import formatter from "../mixins/formatter";
import { CO2_TYPE } from "../units";
import TariffChart from "./TariffChart.vue";

export default {
	name: "TargetChargePlan",
	components: { TariffChart },
	mixins: [formatter],
	props: {
		duration: Number,
		rates: Array,
		plan: Array,
		smartCostType: String,
		targetTime: Date,
		currency: String,
	},
	data() {
		return { activeIndex: null, startTime: new Date() };
	},
	computed: {
		planDuration() {
			return this.fmtDuration(this.duration);
		},
		isCo2() {
			return this.smartCostType === CO2_TYPE;
		},
		hasTariff() {
			return this.rates?.length > 1;
		},
		avgPrice() {
			let hourSum = 0;
			let priceSum = 0;
			this.convertDates(this.plan).forEach((slot) => {
				const hours = (slot.end.getTime() - slot.start.getTime()) / 3600000;
				if (slot.price) {
					hourSum += hours;
					priceSum += hours * slot.price;
				}
			});
			return hourSum ? priceSum / hourSum : undefined;
		},
		fmtAvgPrice() {
			let price = this.activeSlot ? this.activeSlot.price : this.avgPrice;
			if (price === undefined) {
				return this.$t("main.targetChargePlan.unknownPrice");
			}
			return this.isCo2
				? this.fmtCo2Medium(price)
				: this.fmtPricePerKWh(price, this.currency);
		},
		activeSlot() {
			return this.slots[this.activeIndex];
		},
		activeSlotName() {
			if (this.activeSlot) {
				const { day, startHour, endHour } = this.activeSlot;
				const range = `${startHour}–${endHour}`;
				return this.$t("main.targetChargePlan.timeRange", { day, range });
			}
			return null;
		},
		slots() {
			const result = [];
			const rates = this.convertDates(this.rates);
			const plan = this.convertDates(this.plan);
			const oneHour = 60 * 60 * 1000;
			for (let i = 0; i < 42; i++) {
				const start = new Date(this.startTime.getTime() + oneHour * i);
				const startHour = start.getHours();
				start.setMinutes(0);
				start.setSeconds(0);
				start.setMilliseconds(0);
				const end = new Date(start.getTime());
				end.setHours(startHour + 1);
				const endHour = end.getHours();
				const day = this.weekdayShort(start);
				const toLate = this.targetTime.getTime() <= start.getTime();
				// TODO: handle multiple matching time slots
				const price = this.findSlotInRange(start, end, rates)?.price;
				const charging = this.findSlotInRange(start, end, plan) != null;
				result.push({ day, price, startHour, endHour, charging, toLate });
			}
			return result;
		},
	},
	watch: {
		rates() {
			this.startTime = new Date();
		},
	},
	methods: {
		convertDates(list) {
			if (!list?.length) {
				return [];
			}
			return list.map((item) => {
				return {
					start: new Date(item.start),
					end: new Date(item.end),
					price: item.price,
				};
			});
		},
		findSlotInRange(start, end, slots) {
			return slots.find((s) => {
				if (s.start.getTime() < start.getTime()) {
					return s.end.getTime() > start.getTime();
				}
				return s.start.getTime() < end.getTime();
			});
		},
		slotHovered(index) {
			this.activeIndex = index;
		},
	},
};
</script>

<style scoped>
.value {
	font-size: 18px;
	font-weight: bold;
}
.label {
	color: var(--evcc-gray);
	text-transform: uppercase;
}
</style>
