<template>
	<div class="chart">
		<div
			v-for="(slot, index) in slots"
			:key="slot.start"
			:data-index="index"
			class="slot user-select-none"
			:class="{
				active: slot.charging,
				hovered: activeIndex === index,
				toLate: slot.toLate,
				warning: slot.warning,
				'cursor-pointer': slot.selectable,
				faded: activeIndex !== null && activeIndex !== index,
			}"
			@touchstart="hoverSlot(index)"
			@mouseenter="hoverSlot(index)"
			@touchend="hoverSlot(null)"
			@mouseleave="hoverSlot(null)"
			@mouseup="hoverSlot(null)"
			@click="selectSlot(index)"
		>
			<div class="slot-bar" :style="priceStyle(slot.price)">
				<span v-if="slot.price === undefined && avgPrice" class="unknown">?</span>
			</div>
			<div class="slot-label">
				{{ slot.startHour }}
				<br />
				<span v-if="slot.startHour === 0">{{ slot.day }}</span>
			</div>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";
import { CO2_TYPE } from "../units";

export default {
	name: "TariffChart",
	mixins: [formatter],
	props: {
		slots: Array,
	},
	emits: ["slot-hovered", "slot-selected"],
	data() {
		return { activeIndex: null, startTime: new Date() };
	},
	computed: {
		priceInfo() {
			let max = Number.MIN_VALUE;
			let min = 0;
			this.slots
				.map((s) => s.price)
				.filter((price) => price !== undefined)
				.forEach((price) => {
					max = Math.max(max, price);
					min = Math.min(min, price);
				});
			return { min, range: max - min };
		},
		avgPrice() {
			let sum = 0;
			let count = 0;
			this.slots.forEach((s) => {
				if (s.price !== undefined) {
					sum += s.price;
					count++;
				}
			});
			return sum / count;
		},
		isCo2() {
			return this.unit === CO2_TYPE;
		},
		activeSlot() {
			return this.slots[this.activeIndex];
		},
	},
	methods: {
		hoverSlot(index) {
			this.activeIndex = index;
			this.$emit("slot-hovered", index);
		},
		selectSlot(index) {
			if (this.slots[index]?.selectable) {
				this.$emit("slot-selected", index);
			}
		},
		priceStyle(price) {
			const value = price === undefined ? this.avgPrice : price;
			const height =
				value !== undefined && !isNaN(value)
					? `${10 + (90 / this.priceInfo.range) * (value - this.priceInfo.min)}%`
					: "75%";
			return { height };
		},
	},
};
</script>

<style scoped>
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
	transition-property: opacity, background, color;
	transition-duration: var(--evcc-transition-fast);
	transition-timing-function: ease-in;
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
	transition: height var(--evcc-transition-fast) ease-in;
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
.slot.warning .slot-bar {
	background: var(--bs-warning);
}
.slot.warning .slot-label {
	color: var(--bs-warning);
}
.unknown {
	margin: 0 -0.5rem;
}
.slot.hovered {
	opacity: 1;
}
.slot.faded {
	opacity: 0.33;
}
</style>
