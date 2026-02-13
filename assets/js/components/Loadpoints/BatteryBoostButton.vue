<template>
	<button
		class="root d-flex align-items-center justify-content-center position-relative"
		:class="{ active, belowLimit }"
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
			<BatteryBoost />
		</div>
		<div class="icon-wrapper text-white" :style="iconActiveStyle">
			<BatteryBoost />
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
		iconActiveStyle() {
			return {
				display: this.active ? "flex" : "none",
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
	opacity: 1;
}
.root:disabled {
	color: inherit;
	opacity: 0.25;
}
.root:active {
	box-shadow: 0 0 3px 0 #0ba63199;
}
.root.belowLimit:not(:disabled) {
	opacity: 0.5;
}
.root:before,
.root:after {
	content: "";
	position: absolute;
	inset: 0;
	border-radius: var(--bs-border-radius);
	border-width: 1px;
	border-style: solid;
	transition: border-color var(--evcc-transition-very-fast) linear;
}
.root:before {
	border-color: #0ba63133;
	clip-path: inset(0 0 calc(var(--soc)) 0);
}
.root:after {
	border-color: var(--bs-primary);
	clip-path: inset(calc(100% - var(--soc)) 0 0 0);
}
.root:hover:before {
	border-color: #0ba63166;
}
.root.active:after {
	display: none;
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
}
</style>
