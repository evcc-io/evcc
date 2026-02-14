<template>
	<button
		class="root position-relative"
		:class="{ active, belowLimit, full }"
		:style="{ '--soc': `${adjustedSoc}%` }"
		:disabled="disabled"
		data-testid="battery-boost-button"
		@click="toggle"
	>
		<div
			v-if="active"
			class="progress position-absolute"
			:style="{ height: `${adjustedSoc}%` }"
		>
			<div class="progress-bar bg-primary progress-bar-striped progress-bar-animated"></div>
		</div>
		<div class="icon-wrapper" :style="iconStyle">
			<BatteryBoost :active="active && !belowLimit" />
		</div>
		<div class="icon-wrapper text-white" :style="iconActiveStyle">
			<BatteryBoost :active="active && !belowLimit" />
		</div>
	</button>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { CHARGE_MODE } from "@/types/evcc";
import BatteryBoost from "../MaterialIcon/BatteryBoost.vue";

export default defineComponent({
	name: "BatteryBoostButton",
	components: {
		BatteryBoost,
	},
	props: {
		batteryBoost: Boolean,
		batteryBoostLimit: { type: Number, default: 100 },
		mode: String as PropType<CHARGE_MODE>,
		batterySoc: { type: Number, default: 0 },
	},
	emits: ["updated"],
	computed: {
		disabled() {
			return this.mode && [CHARGE_MODE.OFF, CHARGE_MODE.NOW].includes(this.mode);
		},
		adjustedSoc(): number {
			const range = 100 - this.batteryBoostLimit;
			if (range <= 0) return 0;
			return Math.max(
				0,
				Math.min(100, ((this.batterySoc - this.batteryBoostLimit) / range) * 100)
			);
		},
		belowLimit(): boolean {
			return this.batterySoc < this.batteryBoostLimit;
		},
		iconStyle() {
			return {
				clipPath: this.active ? `inset(0 0 calc(var(--soc)) 0)` : undefined,
			};
		},
		active(): boolean {
			return this.batteryBoost;
		},
		full(): boolean {
			return !this.active && this.adjustedSoc >= 90;
		},
		iconActiveStyle() {
			return {
				opacity: this.active ? 1 : 0,
				clipPath: this.active ? `inset(calc(100% - var(--soc)) 0 0 0)` : undefined,
			};
		},
	},
	methods: {
		toggle() {
			this.$emit("updated", !this.batteryBoost);
		},
	},
});
</script>

<style scoped>
.root {
	--size: 32px;
	height: var(--size);
	width: var(--size);
	border-radius: var(--bs-border-radius);
	overflow: hidden;
	border: none;
	background: none;
	padding: 0;
	color: var(--evcc-default-text);
	opacity: 1;
}
.root:disabled {
	color: inherit;
	opacity: 0.25;
}
.root:active {
	box-shadow: 0 0 1px 0 var(--bs-primary);
}
.root.belowLimit:not(:disabled) {
	opacity: 0.5;
}
.root:before,
.root:after {
	content: "";
	position: absolute;
	inset: 0;
	border-color: var(--bs-primary);
	border-radius: var(--bs-border-radius);
	border-width: 2px;
	border-style: solid;
	transition: opacity var(--evcc-transition-very-fast) linear;
}
.root:before {
	opacity: 0.25;
	clip-path: inset(0 0 calc(var(--soc)) 0);
}
.root:after {
	clip-path: inset(calc(100% - var(--soc)) 0 0 0);
}
.root:hover:before {
	opacity: 0.5;
}
.root.active:after {
	display: none;
}
.root.full:after {
	clip-path: none;
	background: conic-gradient(
		from var(--border-angle, 0deg),
		var(--bs-primary-dim) 0%,
		var(--bs-primary) 12%,
		var(--bs-primary-dim) 33%,
		var(--bs-primary) 45%,
		var(--bs-primary-dim) 66%,
		var(--bs-primary) 78%,
		var(--bs-primary-dim) 100%
	);
	border: none;
	mask:
		linear-gradient(#fff 0 0) content-box,
		linear-gradient(#fff 0 0);
	mask-composite: exclude;
	padding: 2px;
	animation: rotate-border 3s linear infinite;
}
.root.full:hover:after,
.root.full:active:after {
	background: var(--bs-primary);
	animation: none;
}
@keyframes rotate-border {
	to {
		--border-angle: 360deg;
	}
}
@property --border-angle {
	syntax: "<angle>";
	initial-value: 0deg;
	inherits: false;
}
.progress {
	border-radius: 0;
	bottom: 0;
	left: 0;
	right: 0;
}
.progress-bar {
	height: 100%;
	width: 100%;
}
.icon-wrapper {
	position: absolute;
	inset: 0;
	display: flex;
	align-items: center;
	justify-content: center;
	transform: translateZ(0); /* fix Safari hover jump */
	transition: opacity var(--evcc-transition-fast) ease;
}
</style>
