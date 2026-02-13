<template>
	<button
		class="root d-flex align-items-center justify-content-center position-relative"
		:class="{ active, full }"
		:style="{ '--soc': `${batterySoc}%` }"
		:disabled="disabled"
		data-testid="battery-boost-button"
		@click="toggle"
	>
		<div v-if="active" class="progress position-absolute" :style="{ height: `${batterySoc}%` }">
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
		mode: String as PropType<CHARGE_MODE>,
		batterySoc: { type: Number, default: 0 },
	},
	emits: ["updated"],
	computed: {
		disabled() {
			return this.mode && [CHARGE_MODE.OFF, CHARGE_MODE.NOW].includes(this.mode);
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
			return this.batterySoc > 95;
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
	box-shadow: 0 0 3px 0 #0ba63133;
	opacity: 1;
}
.root:disabled {
	color: inherit;
	opacity: 0.25;
}
.root:hover {
	box-shadow: 0 0 3px 0 #0ba63166;
}
.root:active {
	box-shadow: 0 0 3px 0 #0ba63199;
}
.root:after {
	content: "";
	position: absolute;
	inset: 0;
	border-radius: var(--bs-border-radius);
	border: 1px solid var(--bs-primary);
	transition: border-color var(--evcc-transition-very-fast) linear;
	clip-path: inset(calc(100% - var(--soc)) 0 0 0);
}
.root.full:after {
	border-width: 2px;
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
