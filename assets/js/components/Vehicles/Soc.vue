<template>
	<div class="vehicle-soc">
		<div class="progress">
			<template v-if="heating">
				<div v-if="connected" class="thermal-track">
					<div
						v-if="
							heatingHasTemp &&
							enabled &&
							remainingSocWidth !== null &&
							remainingSocWidth > 0
						"
						class="thermal-fill thermal-fill--remaining"
						:style="{ width: `${thermalLimitWidth}%`, ...transition }"
					>
						<div
							class="thermal-fill-inner"
							:style="{ width: `${thermalRemainingInnerWidth}%` }"
						></div>
					</div>
					<div
						v-if="heatingHasTemp"
						class="thermal-fill"
						role="progressbar"
						:style="{
							width: `${vehicleSocDisplayWidth}%`,
							opacity: charging ? 1 : 0.55,
							...transition,
						}"
					>
						<div
							class="thermal-fill-inner"
							:style="{ width: `${thermalInnerWidth}%` }"
						></div>
						<div v-if="charging" class="heating-stripes"></div>
					</div>
					<div
						v-else
						class="thermal-fill thermal-fill--warm"
						role="progressbar"
						:style="{ opacity: charging ? 1 : 0.55 }"
					>
						<div v-if="charging" class="heating-stripes"></div>
					</div>
				</div>
			</template>
			<template v-else>
				<div
					v-if="connected"
					class="progress-bar"
					role="progressbar"
					:class="{
						[progressColor]: true,
						'progress-bar-striped': charging,
						'progress-bar-animated': charging,
					}"
					:style="{ width: `${vehicleSocDisplayWidth}%`, ...transition }"
				></div>
				<div
					v-if="
						remainingSocWidth !== null && remainingSocWidth > 0 && enabled && connected
					"
					class="progress-bar bg-muted"
					role="progressbar"
					:class="progressColor"
					:style="{ width: `${remainingSocWidth}%`, ...transition }"
				></div>
			</template>
			<div
				v-show="vehicleLimitSoc"
				ref="vehicleLimitSoc"
				class="vehicle-limit-soc"
				data-bs-toggle="tooltip"
				title=" "
				:class="{ 'vehicle-limit-soc--active': vehicleLimitSocActive }"
				:style="{ left: `${vehicleLimitSocPosition}%` }"
			/>
			<div
				v-show="energyLimitMarkerPosition"
				class="energy-limit-marker"
				data-bs-toggle="tooltip"
				:class="{
					'energy-limit-marker--active': energyLimitMarkerActive,
					'energy-limit-marker--visible':
						energyLimitMarkerPosition !== null && energyLimitMarkerPosition < 100,
				}"
				:style="{ left: `${energyLimitMarkerPosition}%` }"
			/>
			<div
				v-show="planMarkerAvailable"
				class="plan-marker"
				data-bs-toggle="tooltip"
				:style="{ left: `${planMarkerPosition}%` }"
				data-testid="plan-marker"
				@click="$emit('plan-clicked')"
			>
				<shopicon-regular-clock></shopicon-regular-clock>
			</div>
		</div>
		<div class="target">
			<input
				v-if="socBasedCharging && connected"
				type="range"
				:min="lowerBound"
				:max="upperBound"
				:step="step"
				:value="visibleLimitSoc"
				class="slider"
				:class="{ 'slider--active': sliderActive, 'slider--heating': heating }"
				:style="{ '--thumb-pos': `${limitPosition}%` }"
				@mousedown="changeLimitSocStart"
				@touchstart="changeLimitSocStart"
				@input="movedLimitSoc"
				@mouseup="changeLimitSocEnd"
				@touchend="changeLimitSocEnd"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import Tooltip from "bootstrap/js/dist/tooltip";
import "@h2d2/shopicons/es/regular/clock";
import formatter from "@/mixins/formatter";
import { defineComponent, type PropType } from "vue";
import type { LoadpointUi } from "@/types/evcc";

export default defineComponent({
	name: "VehicleSoc",
	mixins: [formatter],
	props: {
		connected: Boolean,
		vehicleSoc: { type: Number, default: 0 },
		vehicleLimitSoc: { type: Number, default: 0 },
		enabled: Boolean,
		charging: Boolean,
		heating: Boolean,
		ui: Object as PropType<LoadpointUi>,
		minSoc: { type: Number, default: 0 },
		minSocNotReached: Boolean,
		effectivePlanSoc: { type: Number, default: 0 },
		effectiveLimitSoc: Number,
		limitEnergy: { type: Number, default: 0 },
		planEnergy: { type: Number, default: 0 },
		chargedEnergy: { type: Number, default: 0 },
		socBasedCharging: Boolean,
		socBasedPlanning: Boolean,
	},
	emits: ["limit-soc-drag", "limit-soc-updated", "plan-clicked"],
	data() {
		return {
			selectedLimitSoc: undefined as number | undefined,
			interactionStartScreenY: null,
			tooltip: null as Tooltip | null,
			dragging: false,
		};
	},
	computed: {
		step() {
			return this.heating ? 1 : 5;
		},
		minTemp() {
			return this.ui?.minTemp ?? 0;
		},
		maxTemp() {
			return this.ui?.maxTemp ?? 100;
		},
		lowerBound() {
			if (!this.heating) {
				return 0;
			}
			const base = this.maxTemp > this.minTemp ? this.minTemp : 0;
			// 0 means no reading yet, don't collapse the scale to it
			return this.vehicleSoc > 0 ? Math.min(base, this.vehicleSoc) : base;
		},
		upperBound() {
			if (!this.heating) {
				return 100;
			}
			const base = this.maxTemp > this.minTemp ? this.maxTemp : 100;
			return this.vehicleSoc > 0 ? Math.max(base, this.vehicleSoc) : base;
		},
		thermalInnerWidth() {
			// widen the inner gradient so colors stay anchored to the full track, not the fill width
			const tp = this.vehicleSocDisplayWidth;
			return tp > 0.5 ? 10000 / tp : 20000;
		},
		thermalLimitWidth() {
			// current temp position plus the remaining span up to the limit
			return this.vehicleSocDisplayWidth + (this.remainingSocWidth ?? 0);
		},
		thermalRemainingInnerWidth() {
			// same anchoring trick as thermalInnerWidth, relative to the limit width
			const tp = this.thermalLimitWidth;
			return tp > 0.5 ? 10000 / tp : 20000;
		},
		limitPosition() {
			return this.toPercent(this.visibleLimitSoc);
		},
		heatingHasTemp() {
			return this.socBasedCharging && this.vehicleSoc > 0;
		},
		vehicleLimitSocPosition() {
			return this.toPercent(this.vehicleLimitSoc);
		},
		vehicleSocDisplayWidth() {
			if (this.socBasedCharging) {
				if (this.vehicleSoc >= 0) {
					return this.toPercent(this.vehicleSoc);
				}
				return 100;
			} else {
				if (this.maxEnergy) {
					return (100 / this.maxEnergy) * (this.chargedEnergy / 1e3);
				}
				return 100;
			}
		},
		transition() {
			if (this.dragging) {
				return { transition: "none" };
			}
			return { transition: "width var(--evcc-transition-fast) linear" };
		},
		maxEnergy() {
			return Math.max(this.planEnergy, this.limitEnergy, this.chargedEnergy / 1e3);
		},
		vehicleLimitSocActive() {
			return this.vehicleLimitSoc > 0 && this.vehicleLimitSoc > this.vehicleSoc;
		},
		planMarkerPosition(): number {
			if (this.socBasedPlanning) {
				return this.toPercent(this.effectivePlanSoc);
			}
			const maxEnergy = Math.max(this.planEnergy, this.limitEnergy);
			if (maxEnergy) {
				return (100 / maxEnergy) * this.planEnergy;
			}
			return 0;
		},
		planMarkerAvailable() {
			if (this.socBasedCharging && !this.socBasedPlanning) {
				// mixed mode (% limit and kWh plan): hide marker
				return false;
			}
			return this.planMarkerPosition > 0;
		},
		energyLimitMarkerPosition() {
			if (this.socBasedCharging) {
				return null;
			}
			if (this.limitEnergy) {
				return (100 / this.maxEnergy) * this.limitEnergy;
			}
			return 100;
		},
		energyLimitMarkerActive() {
			if (this.socBasedCharging) {
				return false;
			}
			if (this.planEnergy) {
				return this.limitEnergy >= this.planEnergy;
			}
			return true;
		},
		sliderActive() {
			const isBelowVehicleLimit = this.visibleLimitSoc <= (this.vehicleLimitSoc || 100);
			const isAbovePlanLimit = this.visibleLimitSoc >= (this.effectivePlanSoc || 0);
			return isBelowVehicleLimit && isAbovePlanLimit;
		},
		progressColor() {
			if (this.minSocNotReached) {
				return "bg-danger";
			}
			return "bg-primary";
		},
		remainingSocWidth() {
			if (this.socBasedCharging) {
				if (this.vehicleSocDisplayWidth === 100) {
					return null;
				}
				if (this.minSocNotReached) {
					return this.toPercent(this.minSoc) - this.vehicleSocDisplayWidth;
				}
				const limit = Math.min(
					this.vehicleLimitSoc || 100,
					Math.max(this.visibleLimitSoc, this.effectivePlanSoc || 0)
				);
				if (limit > this.vehicleSoc) {
					return this.toPercent(limit) - this.vehicleSocDisplayWidth;
				}
			} else {
				return 100 - this.vehicleSocDisplayWidth;
			}

			return null;
		},
		visibleLimitSoc() {
			return Number(this.selectedLimitSoc || this.effectiveLimitSoc);
		},
	},
	watch: {
		effectiveLimitSoc() {
			this.selectedLimitSoc = this.effectiveLimitSoc;
		},
		vehicleLimitSoc() {
			this.updateTooltip();
		},
	},
	mounted() {
		this.updateTooltip();
	},
	methods: {
		toPercent(value: number) {
			const span = this.upperBound - this.lowerBound;
			if (span <= 0) {
				return 0;
			}
			const pct = ((value - this.lowerBound) / span) * 100;
			return Math.min(100, Math.max(0, pct));
		},
		changeLimitSocStart(e: Event) {
			this.dragging = true;
			e.stopPropagation();
		},
		changeLimitSocEnd(e: Event) {
			this.dragging = false;
			const value = parseInt((e.target as HTMLInputElement).value, 10);
			// value changed
			if (value !== this.effectiveLimitSoc) {
				this.$emit("limit-soc-updated", value);
			}
		},
		movedLimitSoc(e: Event) {
			const value = parseInt((e.target as HTMLInputElement).value, 10);
			e.stopPropagation();
			const minLimit = this.heating ? this.lowerBound : 20;
			if (value < minLimit) {
				(e.target as HTMLInputElement).value = minLimit.toString();
				this.selectedLimitSoc = value;
				e.preventDefault();
				return false;
			}
			this.selectedLimitSoc = value;

			this.$emit("limit-soc-drag", this.selectedLimitSoc);
			return true;
		},
		updateTooltip() {
			if (!this.tooltip) {
				this.tooltip = new Tooltip(this.$refs["vehicleLimitSoc"] as HTMLElement);
			}
			const value = this.heating
				? this.fmtTemperature(this.vehicleLimitSoc)
				: this.fmtPercentage(this.vehicleLimitSoc);
			const key = this.heating ? "heatingStatus" : "vehicleStatus";
			const content = `${this.$t(`main.${key}.vehicleLimit`)}: ${value}`;
			this.tooltip.setContent({ ".tooltip-inner": content });
		},
	},
});
</script>
<style scoped>
.vehicle-soc {
	--height: 32px;
	--thumb-overlap: 6px;
	--thumb-width: 12px;
	--label-height: 26px;
	position: relative;
	height: var(--height);
	/* 100cqw on the thumb gradient resolves to the bar width */
	container-type: inline-size;
}
.progress {
	position: relative;
	height: 100%;
	font-size: 1rem;
	background: var(--evcc-background);
}
.progress-bar.bg-muted {
	opacity: 0.5;
}
.thermal-track {
	position: absolute;
	inset: 0;
	overflow: hidden;
	border-radius: inherit;
}
.thermal-fill {
	position: absolute;
	left: 0;
	top: 0;
	bottom: 0;
	overflow: hidden;
}
.thermal-fill-inner {
	position: absolute;
	left: 0;
	top: 0;
	bottom: 0;
	background: var(--evcc-heating-gradient);
}
/* muted gradient slice from current temp to the limit, behind the main fill */
.thermal-fill--remaining {
	opacity: 0.3;
}
/* no reading: full-width bar showing only the warm (right) half of the scale */
.thermal-fill--warm {
	right: 0;
	background: var(--evcc-heating-gradient);
	background-size: 200% 100%;
	background-position: right;
}
.heating-stripes {
	position: absolute;
	inset: 0;
	background-image: linear-gradient(
		-45deg,
		rgba(255, 255, 255, 0.24) 25%,
		transparent 25%,
		transparent 50%,
		rgba(255, 255, 255, 0.24) 50%,
		rgba(255, 255, 255, 0.24) 75%,
		transparent 75%,
		transparent
	);
	background-size: 18px 18px;
	animation: stripeShift 0.7s linear infinite;
}
@keyframes stripeShift {
	from {
		background-position: 0 0;
	}
	to {
		background-position: 18px 0;
	}
}
@media (prefers-reduced-motion: reduce) {
	.heating-stripes {
		animation: none;
	}
}
.bg-light {
	color: var(--bs-gray-dark);
}
.slider {
	-webkit-appearance: none;
	position: absolute;
	top: calc(var(--thumb-overlap) * -1);
	height: calc(100% + 2 * var(--thumb-overlap));
	width: 100%;
	background: transparent;
	pointer-events: none;
}
.slider:focus {
	outline: none;
}
/* Note: Safari,Chrome,Blink and Firefox specific styles need to be in separate definitions to work */
.slider::-webkit-slider-runnable-track {
	position: relative;
	background: transparent;
	border: none;
	height: 100%;
	cursor: auto;
	/* ensure slider start/end is at the edge and not inside the track */
	margin: 0 -6px;
}
.slider::-moz-range-track {
	background: transparent;
	border: none;
	height: 100%;
	cursor: auto;
	/* ensure slider start/end is at the edge and not inside the track */
	margin: 0 -6px;
}
.slider::-webkit-slider-thumb {
	-webkit-appearance: none;
	position: relative;
	margin-left: var(--thumb-width) / 2;
	height: 100%;
	width: var(--thumb-width);
	background-color: var(--evcc-gray);
	cursor: grab;
	border: none;
	opacity: 1;
	border-radius: var(--thumb-overlap);
	box-shadow: 0 0 6px var(--evcc-background);
	pointer-events: auto;
	transition: background-color var(--evcc-transition-fast) linear;
}
.slider::-moz-range-thumb {
	position: relative;
	height: 100%;
	width: var(--thumb-width);
	background-color: var(--evcc-gray);
	cursor: grab;
	border: none;
	opacity: 1;
	border-radius: var(--thumb-overlap);
	box-shadow: 0 0 6px var(--evcc-background);
	pointer-events: auto;
	transition: background-color var(--evcc-transition-fast) linear;
}
.slider--active::-webkit-slider-thumb {
	background-color: var(--evcc-dark-green);
}
.slider--active::-moz-range-thumb {
	background-color: var(--evcc-dark-green);
}
/* thumb shows the gradient slice at its position, shifted via --thumb-pos */
.slider--heating.slider--active::-webkit-slider-thumb {
	background: var(--evcc-heating-gradient);
	background-size: 100cqw 100%;
	background-position-x: var(--thumb-pos, 50%);
	background-repeat: no-repeat;
}
.slider--heating.slider--active::-moz-range-thumb {
	background: var(--evcc-heating-gradient);
	background-size: 100cqw 100%;
	background-position-x: var(--thumb-pos, 50%);
	background-repeat: no-repeat;
}
.slider--heating:not(.slider--active)::-webkit-slider-thumb {
	background: var(--evcc-gray);
}
.slider--heating:not(.slider--active)::-moz-range-thumb {
	background: var(--evcc-gray);
}
.vehicle-limit-soc {
	position: absolute;
	top: 0;
	bottom: 0;
	width: 20px;
	transform: translateX(-8px);
	background-color: transparent;
	background-clip: padding-box;
	border-width: 0 8px;
	border-style: solid;
	border-color: transparent;
	transition-property: background-color, left;
	transition-timing-function: linear;
	transition-duration: var(--evcc-transition-fast);
}
.vehicle-limit-soc--active {
	background-color: var(--evcc-box);
}
.plan-marker {
	position: absolute;
	top: 0;
	transform: translateX(-50%);
	color: var(--evcc-darker-green);
	transition-property: color, left, opacity;
	transition-timing-function: linear;
	cursor: pointer;
	transition-duration: var(--evcc-transition-fast);
}
.plan-marker::before {
	content: "";
	display: block;
	height: var(--height);
	border-width: 0 10px;
	background-clip: padding-box;
	border-style: solid;
	border-color: transparent;
	background-color: var(--evcc-darker-green);
	transition: background-color var(--evcc-transition-fast) linear;
}
.energy-limit-marker {
	position: absolute;
	top: -4px;
	bottom: -4px;
	transform: translateX(-50%);
	width: 4px;
	opacity: 0;
	background-color: var(--evcc-gray);
	transition-property: opacity, left, background-color;
	transition-timing-function: linear;
	transition-duration: var(--evcc-transition-fast);
}
.energy-limit-marker--active {
	background-color: var(--evcc-dark-green);
}
.energy-limit-marker--visible {
	opacity: 1;
}
</style>
