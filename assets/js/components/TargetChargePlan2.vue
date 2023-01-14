<template>
	<div class="plan pt-2">
		<div class="details justify-content-between mb-2 d-flex justify-content-between">
			<div class="text-start">
				<div class="label">Ladezeit</div>
				<div class="wert text-primary">{{ planDuration }}</div>
			</div>
			<div v-if="isCo2" class="text-end">
				<div class="label">
					<span v-if="activeSlot">{{ activeSlotName }}</span>
					<span v-else>CO₂-Menge Ø</span>
				</div>
				<div class="wert text-primary">{{ avgCo2 }}</div>
			</div>
			<div v-else class="text-end">
				<div class="label">Energiepreis</div>
				<div class="wert text-primary">{{ avgPrice }}</div>
			</div>
		</div>
		<div class="chart">
			<div
				v-for="(slot, index) in slots"
				:key="slot.start"
				:data-index="index"
				class="slot user-select-none"
				:class="{ active: isActive(slot), behind: slot.behind }"
				@touchstart="activeIndex = index"
				@mouseenter="activeIndex = index"
				@touchmove="touchmove"
				@touchend="activeIndex = null"
				@mouseleave="activeIndex = null"
				@mouseup="activeIndex = null"
			>
				<div
					class="slot-bar"
					:style="priceStyle(slot.price)"
					:title="fmtPricePerKWh(slot.price, plan.unit)"
				></div>
				<div class="slot-label">{{ slot.hour }}<br />{{ slot.day }}</div>
			</div>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/clock";

function sortByPrice(a, b) {
	if (a.price < b.price) {
		return -1;
	}
	if (a.price > b.price) {
		return 1;
	}
	return 0;
}

export default {
	name: "TargetChargePlan2",
	mixins: [formatter],
	props: {
		duration: Number,
		rates: Array,
		plan: Array,
		unit: String,
		targetTime: Date,
	},
	data() {
		return { activeIndex: null };
	},
	computed: {
		planDuration() {
			return this.fmtShortDuration(this.duration, true);
		},
		maxPrice() {
			let result = 0;
			this.slots.forEach(({ price }) => {
				result = Math.max(result, price);
			});
			return result;
		},
		slots() {
			const maxItems = 24 * 1;
			const oneHour = 60 * 60 * 1000;
			const slots = [];
			const now = new Date().getTime();
			for (let i = 0; i < maxItems; i++) {
				const start = new Date(now + oneHour * i);
				const hour = start.getHours();
				const end = new Date(start.getTime());
				end.setHours(hour + 1);
				const day = hour === 0 ? this.weekdayShort(start) : "";
				const behind = this.targetTime.getTime() < start.getTime();
				const price = this.ratePrice(start, end); //Math.round((hour < 5 ? 100 : 400) + Math.random() * 300);
				slots.push({ hour, day, price, start, end, used: false, behind });
			}
			const slotsCopy = [...slots];
			slotsCopy.sort(sortByPrice);
			slotsCopy.forEach((slot, i) => {
				slot.used = i < 3;
			});
			return slots;
		},
		isCo2() {
			return this.unit === "gCO2eq";
		},
		avgCo2() {
			return `${this.activeSlot ? this.activeSlot.price : "301"} g/kWh`;
		},
		avgPrice() {
			return this.fmtPricePerKWh(0.32, this.unit);
		},
		activeSlot() {
			return this.slots[this.activeIndex];
		},
		activeSlotName() {
			if (this.activeSlot) {
				const { start, end } = this.activeSlot;
				return `${start.getHours()}–${end.getHours()} Uhr`;
			}
			return null;
		},
	},
	methods: {
		ratePrice(start, end) {
			return this.rates?.find((r) => {
				const rStart = new Date(r.start).getTime();
				return start.getTime() <= rStart && rStart < end.getTime();
			}).price;
		},
		isActive(slot) {
			return this.activeSlot ? this.activeSlot.start === slot.start : slot.used;
		},
		priceStyle(price) {
			return {
				height: `${(100 / this.maxPrice) * price}%`,
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
	height: 95px;
	align-items: flex-end;
	overflow-y: none;
	overflow-x: auto;
	padding-bottom: 45px;
}
.slot {
	width: 20px;
	flex-grow: 0;
	flex-shrink: 0;
	text-align: center;
	padding: 4px;
	height: 100%;
	display: flex;
	justify-content: flex-end;
	flex-direction: column;
	position: relative;
	opacity: 1;
}

.slot-bar {
	background-clip: content-box !important;
	background: var(--bs-gray-light);
	border-radius: 8px;
	width: 100%;
}
.slot-label {
	color: var(--bs-gray-light);
	line-height: 1.1;
	position: absolute;
	top: 100%;
	left: 0;
	width: 100%;
	text-align: center;
}
.slot.active .slot-bar {
	background: var(--bs-primary);
}
.slot.active .slot-label {
	color: var(--bs-primary);
}
.slot.behind {
	opacity: 0.33;
}
.slot.active {
	opacity: 1;
}
.wert {
	font-size: 18px;
	font-weight: bold;
}
.label {
	color: var(--evcc-gray);
	text-transform: uppercase;
}
</style>
