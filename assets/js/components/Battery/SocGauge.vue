<template>
	<span class="position-relative d-inline-flex flex-shrink-0">
		<svg :width="SIZE" :height="SIZE" :viewBox="`0 0 ${SIZE} ${SIZE}`">
			<circle
				:cx="C"
				:cy="C"
				:r="R"
				fill="none"
				stroke="var(--bs-border-color)"
				:stroke-width="STROKE"
			/>
			<circle
				:cx="C"
				:cy="C"
				:r="R"
				fill="none"
				:stroke="color"
				:stroke-width="STROKE"
				stroke-linecap="round"
				:stroke-dasharray="CIRCUMFERENCE"
				:stroke-dashoffset="offset"
				:transform="`rotate(-90 ${C} ${C})`"
				class="progress"
			/>
		</svg>
		<span class="status-icon" :style="{ color }">
			<shopicon-regular-powersupply
				size="s"
				class="layer grid-icon"
				:class="{ 'layer--active': mode === BATTERY_MODE.CHARGE }"
			></shopicon-regular-powersupply>
			<ArrowDown
				:size="ICON_SIZE.M"
				class="layer"
				:class="{ 'layer--active': isCharging || isDischarging }"
				:style="{ '--rotate': arrowRotate }"
			/>
			<BatteryHold
				:size="ICON_SIZE.M"
				class="layer"
				:class="{ 'layer--active': mode === BATTERY_MODE.HOLD }"
			/>
			<BatteryHoldCharge
				:size="ICON_SIZE.M"
				class="layer"
				:class="{ 'layer--active': mode === BATTERY_MODE.HOLDCHARGE }"
			/>
			<Dot :size="ICON_SIZE.S" class="layer" :class="{ 'layer--active': isIdle }" />
		</span>
	</span>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import "@h2d2/shopicons/es/regular/powersupply";
import { BATTERY_MODE, ICON_SIZE } from "@/types/evcc";
import ArrowDown from "../MaterialIcon/ArrowDown.vue";
import BatteryHold from "../MaterialIcon/BatteryHold.vue";
import BatteryHoldCharge from "../MaterialIcon/BatteryHoldCharge.vue";
import Dot from "../MaterialIcon/Dot.vue";

// fixed ring geometry, computed once rather than per instance
const SIZE = 38;
const STROKE = Math.max(3, Math.round(SIZE * 0.1));
const C = SIZE / 2;
const R = (SIZE - STROKE) / 2;
const CIRCUMFERENCE = 2 * Math.PI * R;

const LOCKED_MODES: BATTERY_MODE[] = [
	BATTERY_MODE.HOLD,
	BATTERY_MODE.HOLDCHARGE,
	BATTERY_MODE.CHARGE,
];

// all four icon layers stay mounted and cross-fade via opacity/scale, so switching state
// tweens instead of hard-swapping
export default defineComponent({
	name: "SocGauge",
	components: { ArrowDown, BatteryHold, BatteryHoldCharge, Dot },
	props: {
		soc: { type: Number, default: 0 },
		color: { type: String, default: "" },
		power: { type: Number, default: 0 }, // W, + discharging / - charging
		mode: String as PropType<BATTERY_MODE>,
	},
	data() {
		return { ICON_SIZE, BATTERY_MODE, SIZE, STROKE, C, R, CIRCUMFERENCE };
	},
	computed: {
		isLocked(): boolean {
			return LOCKED_MODES.includes(this.mode as BATTERY_MODE);
		},
		isCharging(): boolean {
			return !this.isLocked && this.power < -50;
		},
		isDischarging(): boolean {
			return !this.isLocked && this.power > 50;
		},
		isIdle(): boolean {
			return !this.isLocked && !this.isCharging && !this.isDischarging;
		},
		// rests left-facing so it always swings into place instead of just fading in
		arrowRotate(): string {
			if (this.isCharging) return "180deg";
			if (this.isDischarging) return "0deg";
			return "90deg";
		},
		offset(): number {
			return CIRCUMFERENCE * (1 - Math.max(0, Math.min(100, this.soc)) / 100);
		},
	},
});
</script>

<style scoped>
.progress {
	transition: stroke-dashoffset var(--evcc-transition-medium) ease;
}
.status-icon {
	position: absolute;
	inset: 0;
	display: grid;
	place-items: center;
}
.layer {
	grid-area: 1 / 1;
	opacity: 0;
	--rotate: 0deg;
	--scale: 1;
	transform: scale(calc(var(--scale) * 0.5)) rotate(var(--rotate));
	transition:
		opacity var(--evcc-transition-medium) ease,
		transform var(--evcc-transition-medium) ease;
}
.layer--active {
	opacity: 1;
	transform: scale(var(--scale)) rotate(var(--rotate));
}
.grid-icon {
	--scale: 0.9;
}
</style>
