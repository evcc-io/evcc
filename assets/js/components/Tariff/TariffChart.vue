<template>
	<div class="root position-relative">
		<div class="chart position-relative">
			<div
				v-for="(slot, index) in slots"
				:key="`${slot.day}-${slot.startHour}`"
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
				<div class="slot-bar" :style="valueStyle(slot.value)">
					<span v-if="slot.value === undefined && avgValue" class="unknown">?</span>
				</div>
				<div class="slot-label">
					<span v-if="!slot.isTarget || targetNearlyOutOfRange">{{
						formatHour(slot.startHour)
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

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import "@h2d2/shopicons/es/regular/arrowright";
import { is12hFormat } from "@/units";
import PlanEndIcon from "../MaterialIcon/PlanEnd.vue";
import formatter from "@/mixins/formatter";
import type { Slot } from "@/types/evcc";

const BAR_WIDTH = 20;

export default defineComponent({
	name: "TariffChart",
	components: {
		PlanEndIcon,
	},
	mixins: [formatter],
	props: {
		slots: { type: Array as PropType<Slot[]>, default: () => [] },
		targetText: [String, null],
		targetOffset: { type: Number, default: 0 },
	},
	emits: ["slot-hovered", "slot-selected"],
	data() {
		return {
			activeIndex: null as number | null,
			startTime: new Date(),
			longPressTimer: undefined as number | undefined,
			isLongPress: false,
		};
	},
	computed: {
		valueInfo() {
			let max = Number.MIN_VALUE;
			let min = 0;
			this.slots
				.map((s) => s.value)
				.filter((value) => value !== undefined)
				.forEach((value) => {
					max = Math.max(max, value);
					min = Math.min(min, value);
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
		avgValue() {
			let sum = 0;
			let count = 0;
			this.slots.forEach((s) => {
				if (s.value !== undefined) {
					sum += s.value;
					count++;
				}
			});
			return sum / count;
		},
	},
	methods: {
		hoverSlot(index: number | null) {
			this.activeIndex = index;
			this.$emit("slot-hovered", index);
		},
		selectSlot(index: number) {
			if (this.isLongPress) {
				this.isLongPress = false;
				return;
			}
			if (this.slots[index]?.selectable) {
				this.$emit("slot-selected", index);
			}
		},
		showWeekday(index: number) {
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
		valueStyle(value: number | undefined) {
			const val = value === undefined ? this.avgValue : value;
			const height =
				value !== undefined && !isNaN(val)
					? `${10 + (90 / this.valueInfo.range) * (val - this.valueInfo.min)}%`
					: "75%";
			return { height };
		},
		startLongPress(index: number) {
			this.longPressTimer = setTimeout(() => {
				this.isLongPress = true;
				this.hoverSlot(index);
			}, 300) as unknown as number;
		},
		cancelLongPress() {
			clearTimeout(this.longPressTimer);
			this.hoverSlot(null);
		},
		formatHour(hour: number) {
			return is12hFormat() ? hour % 12 : hour;
		},
	},
});
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
