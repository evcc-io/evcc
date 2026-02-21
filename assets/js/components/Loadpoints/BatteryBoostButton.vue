<template>
	<button
		class="root position-relative"
		tabindex="0"
		:class="{ active, belowLimit, full }"
		:style="{ '--soc': `${adjustedSoc}%` }"
		:disabled="disabled"
		:aria-label="ariaLabel"
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
import { CHARGE_MODE, type Timeout } from "@/types/evcc";
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
	emits: ["updated", "status"],
	data() {
		return {
			selected: null as boolean | null,
			timeout: null as Timeout | null,
		};
	},
	computed: {
		active(): boolean {
			return this.selected ?? this.batteryBoost;
		},
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
		full(): boolean {
			return !this.active && this.adjustedSoc >= 90;
		},
		ariaLabel(): string {
			const t = (key: string) => this.$t(`main.loadpointSettings.batteryBoost.${key}`);
			if (this.active) return t("stateActive");
			if (this.belowLimit) return t("stateBelowLimit");
			return t("stateReady");
		},
		iconActiveStyle() {
			return {
				opacity: this.active ? 1 : 0,
				clipPath: this.active ? `inset(calc(100% - var(--soc)) 0 0 0)` : undefined,
			};
		},
	},
	watch: {
		batteryBoost() {
			this.clearSelected();
		},
	},
	unmounted() {
		this.clearSelected();
	},
	methods: {
		toggle() {
			(this.$el as HTMLElement).blur();
			const status = (key: string, params?: Record<string, unknown>, type?: string) =>
				this.$emit("status", {
					message: this.$t(`main.vehicleStatus.${key}`, params ?? {}),
					type,
				});
			const newValue = !this.active;

			// below limit: only show message, don't toggle
			if (newValue && this.belowLimit) {
				status("batteryBoostBelowLimit");
				return;
			}

			// optimistic update, instant user feedback
			this.selected = newValue;
			if (this.timeout) clearTimeout(this.timeout);
			this.timeout = setTimeout(() => this.clearSelected(), 5000);

			if (newValue) {
				status("batteryBoostEnabled", { limit: `${this.batteryBoostLimit}%` }, "primary");
			} else {
				status("batteryBoostDisabled");
			}
			this.$emit("updated", newValue);
		},
		clearSelected() {
			this.selected = null;
			if (this.timeout) {
				clearTimeout(this.timeout);
				this.timeout = null;
			}
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
	transition: opacity var(--evcc-transition-fast) linear;
}
.root:focus,
.root:active {
	outline: var(--bs-focus-ring-width) solid var(--bs-focus-ring-color);
	outline-width: var(--bs-focus-ring-width);
}
.root:disabled {
	color: inherit;
	opacity: 0.25;
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
	transition: opacity var(--evcc-transition-fast) linear;
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
.root.full:hover:after {
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
	background-image: linear-gradient(
		0deg,
		rgba(255, 255, 255, 0.15) 25%,
		transparent 25%,
		transparent 50%,
		rgba(255, 255, 255, 0.15) 50%,
		rgba(255, 255, 255, 0.15) 75%,
		transparent 75%,
		transparent
	) !important;
	animation: progress-bar-stripes-down 1s linear infinite !important;
}
@keyframes progress-bar-stripes-down {
	from {
		background-position: 0 0;
	}
	to {
		background-position: 0 1rem;
	}
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
