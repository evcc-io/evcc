<template>
	<div class="plan pt-2">
		<div class="details justify-content-between mb-2 d-flex justify-content-between">
			<div class="text-start">
				<div class="label">{{ $t("main.targetChargePlan.chargeDuration") }}</div>
				<div class="value text-primary">{{ planDuration }}</div>
			</div>
			<div class="text-end">
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
		<div class="chart">
			<div
				v-for="(slot, index) in slots"
				:key="slot.start"
				:data-index="index"
				class="slot user-select-none"
				:class="{ active: isActive(index), toLate: slot.toLate }"
				@touchstart="activeIndex = index"
				@mouseenter="activeIndex = index"
				@touchmove="touchmove"
				@touchend="activeIndex = null"
				@mouseleave="activeIndex = null"
				@mouseup="activeIndex = null"
			>
				<div class="slot-bar" :style="priceStyle(slot.price)">
					<span v-if="slot.price === undefined" class="unknown">?</span>
				</div>
				<div class="slot-label">
					{{ slot.startHour }}
					<br />
					<span v-if="slot.startHour === 0">{{ slot.day }}</span>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/clock";

export default {
	name: "TargetChargePlan",
	mixins: [formatter],
	props: {
		duration: Number,
		rates: Array,
		plan: Array,
		unit: String,
		targetTime: Date,
	},
	data() {
		return { activeIndex: null, startTime: new Date() };
	},
	computed: {
		planDuration() {
			return this.fmtShortDuration(this.duration, true);
		},
		maxPrice() {
			let result = 0;
			this.slots
				.map((s) => s.price)
				.filter((price) => price !== undefined)
				.forEach((price) => {
					result = Math.max(result, price);
				});
			return result;
		},
		isCo2() {
			return this.unit === "gCO2eq";
		},
		avgPrice() {
			let hourSum = 0;
			let priceSum = 0;
			this.convertDates(this.plan).forEach((slot) => {
				const hours = (slot.end.getTime() - slot.start.getTime()) / 36e5;
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
				? `${Math.round(price)} g/kWh`
				: this.fmtPricePerKWh(price, this.unit);
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
		ratePrice(start, end) {
			return this.rates?.find((r) => {
				const rStart = new Date(r.start).getTime();
				return start.getTime() <= rStart && rStart < end.getTime();
			}).price;
		},
		isActive(index) {
			return this.activeIndex !== null
				? this.activeIndex === index
				: this.slots[index].charging;
		},
		priceStyle(price) {
			return {
				height: price === undefined ? "100%" : `${(100 / this.maxPrice) * price}%`,
			};
		},
		touchmove(e) {
			let $el = document.elementFromPoint(e.touches[0].clientX, e.touches[0].clientY);
			if (!$el) return;
			if (!$el.classList.contains("slot")) {
				$el = $el.parent;
			}
			if (!$el) return;
			const index = $el.getAttribute("data-index");
			if (!index) return;
			this.activeIndex = index * 1;
		},
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
	},
};
</script>

<style scoped>
.root {
	overflow: hidden;
	position: relative;
}
.chart {
	display: flex;
	height: 140px;
	align-items: flex-end;
	overflow-y: none;
	padding-bottom: 45px;
}
.slot {
	text-align: center;
	padding: 4px;
	height: 100%;
	display: flex;
	justify-content: flex-end;
	flex-direction: column;
	position: relative;
	opacity: 1;
}
@media (max-width: 991px) {
	.chart {
		overflow-x: auto;
	}
	.slot {
		width: 20px;
		flex-grow: 0;
		flex-shrink: 0;
	}
}
@media (min-width: 992px) {
	.chart {
		overflow-x: none;
		justify-content: stretch;
	}
	.slot {
		flex-grow: 1;
		flex-shrink: 1;
		flex-basis: 1;
	}
}
.slot-bar {
	background-clip: content-box !important;
	background: var(--bs-gray-light);
	border-radius: 8px;
	width: 100%;
	align-items: center;
	display: flex;
	justify-content: center;
	color: var(--bs-white);
}
.slot-label {
	color: var(--bs-gray-light);
	line-height: 1.1;
	position: absolute;
	top: 100%;
	left: -50%;
	width: 200%;
	text-align: center;
}
.slot.active .slot-bar {
	background: var(--bs-primary);
}
.slot.active .slot-label {
	color: var(--bs-primary);
}
.slot.toLate {
	opacity: 0.33;
}
.slot.active {
	opacity: 1;
}
.value {
	font-size: 18px;
	font-weight: bold;
}
.label {
	color: var(--evcc-gray);
	text-transform: uppercase;
}
.unknown {
	margin: 0 -0.5rem;
}
</style>
