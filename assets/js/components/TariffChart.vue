<template>
	<div class="root position-relative">
		<div class="chart position-relative">
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
				@touchstart="startLongPress(index)"
				@touchend="cancelLongPress"
				@touchmove="cancelLongPress"
				@mouseenter="hoverSlot(index)"
				@mouseleave="hoverSlot(null)"
				@mouseup="hoverSlot(null)"
				@click="selectSlot(index)"
			>
				<div class="slot-bar" :style="priceStyle(slot.price)">
					<span v-if="slot.price === undefined && avgPrice" class="unknown">?</span>
				</div>
				<div class="slot-label">
					<span v-if="!slot.isTarget || targetNearlyOutOfRange">{{
						slot.startHour
					}}</span>
					<br />
					<span v-if="showWeekday(index)">{{ slot.day }}</span>
				</div>
			</div>
			<div
				v-if="targetText && !targetNearlyOutOfRange"
				class="target-inline"
				:class="{ 'target-inline--faded': activeIndex !== null }"
				:style="{ left: targetLeft }"
			>
				<div class="target-inline-marker"></div>
				<div class="text-nowrap target-inline-text" data-testid="target-text">
					{{ targetText }}
				</div>
			</div>
		</div>
		<div v-if="targetText && targetNearlyOutOfRange" ref="targetFixed" class="target-fixed">
			<div class="text-nowrap" data-testid="target-text">{{ targetText }}</div>
			<PlanEndIcon v-if="targetOutOfRange" />
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/arrowright";
import PlanEndIcon from "./MaterialIcon/PlanEnd.vue";
import formatter from "../mixins/formatter";
import { CO2_TYPE } from "../units";

const BAR_WIDTH = 20;

export default {
	name: "TariffChart",
	components: {
		PlanEndIcon,
	},
	mixins: [formatter],
	props: {
		slots: Array,
		targetText: String,
		targetOffset: Number,
	},
	emits: ["slot-hovered", "slot-selected"],
	data() {
		return {
			activeIndex: null,
			startTime: new Date(),
			longPressTimer: null,
			isLongPress: false,
		};
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
		targetLeft() {
			const fullHours = Math.floor(this.targetOffset);
			const hourFraction = this.targetOffset - fullHours;
			return `${fullHours * BAR_WIDTH + 4 + hourFraction * 12}px`;
		},
		targetNearlyOutOfRange() {
			return this.targetOffset > this.slots.length - 4;
		},
		targetOutOfRange() {
			return this.targetOffset > this.slots.length;
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
			if (this.isLongPress) {
				this.isLongPress = false;
				return;
			}
			if (this.slots[index]?.selectable) {
				this.$emit("slot-selected", index);
			}
		},
		showWeekday(index) {
			const slot = this.slots[index];
			if (!slot) {
				return false;
			}
			for (let i = 0; i < 7; i++) {
				if (this.slots[index - i]?.isTarget) {
					return false;
				}
			}
			if (slot.startHour === 0) {
				return true;
			}
			return false;
		},
		priceStyle(price) {
			const value = price === undefined ? this.avgPrice : price;
			const height =
				value !== undefined && !isNaN(value)
					? `${10 + (90 / this.priceInfo.range) * (value - this.priceInfo.min)}%`
					: "75%";
			return { height };
		},
		startLongPress(index) {
			this.longPressTimer = setTimeout(() => {
				this.isLongPress = true;
				this.hoverSlot(index);
			}, 300);
		},
		cancelLongPress() {
			clearTimeout(this.longPressTimer);
			this.hoverSlot(null);
		},
	},
};
</script>

<style scoped>
.chart {
	display: flex;
	height: 150px;
	overflow-x: auto;
	align-items: flex-end;
	overflow-y: none;
	padding-bottom: 55px;
}
.target-inline {
	height: 130px;
	position: absolute;
	top: 0;
	display: flex;
	align-items: flex-end;
	color: var(--bs-primary);
	opacity: 1;
	transition-property: opacity, color, left;
	transition-duration: var(--evcc-transition-fast);
	pointer-events: none;
	transition-timing-function: ease-in;
}
.target-inline--faded {
	opacity: 0.33;
}
.target-fixed {
	position: absolute;
	background-color: var(--evcc-box);
	color: var(--bs-primary);
	padding-left: 1rem;
	right: 0;
	top: 110px;
	display: flex;
	gap: 0.25rem;
	align-items: center;
}
.target-inline-marker {
	height: 100%;
	border: 1px solid var(--bs-primary);
	border-width: 0 0 1px 1px;
	width: 0.75rem;
	height: 1.5rem;
	margin-right: 0.25rem;
	margin-bottom: 11px;
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
	width: 20px;
	flex-grow: 0;
	flex-shrink: 0;
}
@media (max-width: 991px) {
	.chart {
		overflow-x: auto;
	}
}
@media (min-width: 992px) {
	.chart {
		overflow-x: hidden;
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
