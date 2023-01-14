<template>
	<div class="plan">
		<div class="details justify-content-between mb-2">
			<div>
				<div class="label">CO₂-Menge Ø</div>
				<!--<div class="label">13–14 Uhr</div>-->
				<div class="wert text-primary">230 g/kWh</div>
			</div>
		</div>
		<div class="chart">
			<div v-for="slot in slots" :key="slot.start" class="slot">
				<div
					class="slot-bar"
					:style="priceStyle(slot.price)"
					:title="fmtPricePerKWh(slot.price, plan.unit)"
					:class="{ used: slot.used }"
				></div>
				<div class="slot-label">{{ slot.label }}</div>
			</div>
		</div>
		<div class="details justify-content-between mt-3">
			<div class="d-flex align-items-center">
				<span> <span>Ladedauer: </span> <strong class="">2h 47m </strong> </span>
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
		tariff: Object,
		plan: Object,
		targetTime: String,
	},
	computed: {
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
				const date = new Date(now + oneHour * i);
				const hour = date.getHours();
				const label = hour === 0 ? "Sa" : hour;
				const price = (hour < 5 ? 10 : 40) + Math.random() * 30;
				slots.push({ label, price, start: date, used: false });
			}
			const slotsCopy = [...slots];
			slotsCopy.sort(sortByPrice);
			slotsCopy.forEach((slot, i) => {
				slot.used = i < 3;
			});
			return slots;
		},
	},
	methods: {
		priceStyle(price) {
			return {
				height: `${(100 / this.maxPrice) * price}%`,
			};
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
	height: 80px;
	align-items: flex-end;
}
.slot {
	width: 16px;
	flex-grow: 0;
	flex-shrink: 0;
	margin: 4px;
	margin-bottom: 0;
	text-align: center;
	height: 100%;
	display: flex;
	justify-content: flex-end;
	flex-direction: column;
}

.slot-bar {
	background: var(--bs-gray-light);
	border-radius: 8px;
	width: 100%;
}
.slot-bar.used {
	background: var(--bs-primary);
}
.chart:hover .slot-bar {
	background: var(--bs-gray-light);
}
.chart:hover .slot-bar:hover {
	background: var(--bs-primary);
}
.slot-label {
	color: var(--bs-gray-light);
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
