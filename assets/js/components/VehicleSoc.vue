<template>
	<div class="vehicle-soc">
		<div class="progress">
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
				v-if="remainingSocWidth > 0 && enabled && connected"
				class="progress-bar bg-muted"
				role="progressbar"
				:class="progressColor"
				:style="{ width: `${remainingSocWidth}%`, ...transition }"
			></div>
			<div
				v-show="vehicleTargetSoc"
				ref="vehicleTargetSoc"
				class="vehicle-limit-soc"
				data-bs-toggle="tooltip"
				title=" "
				:class="{ 'vehicle-limit-soc--active': vehicleTargetSocActive }"
				:style="{ left: `${vehicleTargetSoc}%` }"
			/>
			<div
				v-show="energyLimitMarkerPosition"
				class="energy-limit-marker"
				data-bs-toggle="tooltip"
				:class="{
					'energy-limit-marker--active': energyLimitMarkerActive,
					'energy-limit-marker--visible': energyLimitMarkerPosition < 100,
				}"
				:style="{ left: `${energyLimitMarkerPosition}%` }"
			/>
			<div
				v-show="planMarkerAvailable"
				class="plan-marker"
				data-bs-toggle="tooltip"
				:class="{
					'plan-marker--warning': planOverrun,
					'plan-marker--error': planMarkerUnreachable,
				}"
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
				min="0"
				max="100"
				:step="step"
				:value="visibleLimitSoc"
				class="slider"
				:class="{ 'slider--active': sliderActive }"
				@mousedown="changeLimitSocStart"
				@touchstart="changeLimitSocStart"
				@input="movedLimitSoc"
				@mouseup="changeLimitSocEnd"
				@touchend="changeLimitSocEnd"
			/>
		</div>
	</div>
</template>

<script>
import Tooltip from "bootstrap/js/dist/tooltip";
import "@h2d2/shopicons/es/regular/clock";

export default {
	name: "VehicleSoc",
	props: {
		connected: Boolean,
		vehicleSoc: Number,
		vehicleTargetSoc: Number,
		enabled: Boolean,
		charging: Boolean,
		heating: Boolean,
		minSoc: Number,
		effectivePlanSoc: Number,
		effectiveLimitSoc: Number,
		limitEnergy: Number,
		planEnergy: Number,
		planOverrun: Boolean,
		chargedEnergy: Number,
		socBasedCharging: Boolean,
		socBasedPlanning: Boolean,
	},
	emits: ["limit-soc-drag", "limit-soc-updated", "plan-clicked"],
	data: function () {
		return {
			selectedLimitSoc: null,
			interactionStartScreenY: null,
			tooltip: null,
			dragging: false,
		};
	},
	computed: {
		step: function () {
			return this.heating ? 1 : 5;
		},
		vehicleSocDisplayWidth: function () {
			if (this.socBasedCharging) {
				if (this.vehicleSoc >= 0) {
					return this.vehicleSoc;
				}
				return 100;
			} else {
				if (this.maxEnergy) {
					return (100 / this.maxEnergy) * (this.chargedEnergy / 1e3);
				}
				return 100;
			}
		},
		transition: function () {
			if (this.dragging) {
				return { transition: "none" };
			}
			return { transition: "width var(--evcc-transition-fast) linear" };
		},
		maxEnergy: function () {
			return Math.max(this.planEnergy, this.limitEnergy, this.chargedEnergy / 1e3);
		},
		vehicleTargetSocActive: function () {
			return this.vehicleTargetSoc > 0 && this.vehicleTargetSoc > this.vehicleSoc;
		},
		planMarkerPosition: function () {
			if (this.socBasedPlanning) {
				return this.effectivePlanSoc;
			}
			const maxEnergy = Math.max(this.planEnergy, this.limitEnergy);
			if (maxEnergy) {
				return (100 / maxEnergy) * this.planEnergy;
			}
			return 0;
		},
		planMarkerAvailable: function () {
			if (this.socBasedCharging && !this.socBasedPlanning) {
				// mixed mode (% limit and kWh plan): hide marker
				return false;
			}
			return this.planMarkerPosition > 0;
		},
		planMarkerUnreachable: function () {
			if (this.socBasedPlanning) {
				const vehicleLimit = this.vehicleTargetSoc || 100;
				return this.effectivePlanSoc > vehicleLimit;
			}
			return false;
		},
		energyLimitMarkerPosition: function () {
			if (this.socBasedCharging) {
				return null;
			}
			if (this.limitEnergy) {
				return (100 / this.maxEnergy) * this.limitEnergy;
			}
			return 100;
		},
		energyLimitMarkerActive: function () {
			if (this.socBasedCharging) {
				return false;
			}
			if (this.planEnergy) {
				return this.limitEnergy >= this.planEnergy;
			}
			return true;
		},
		sliderActive: function () {
			const isBelowVehicleLimit = this.visibleLimitSoc <= (this.vehicleTargetSoc || 100);
			const isAbovePlanLimit = this.visibleLimitSoc >= (this.effectivePlanSoc || 0);
			return isBelowVehicleLimit && isAbovePlanLimit;
		},
		progressColor: function () {
			if (this.minSocActive) {
				return "bg-danger";
			}
			return "bg-primary";
		},
		minSocActive: function () {
			return this.minSoc > 0 && this.vehicleSoc < this.minSoc;
		},
		remainingSocWidth: function () {
			if (this.socBasedCharging) {
				if (this.vehicleSocDisplayWidth === 100) {
					return null;
				}
				if (this.minSocActive) {
					return this.minSoc - this.vehicleSoc;
				}
				const limit = Math.min(
					this.vehicleTargetSoc || 100,
					Math.max(this.visibleLimitSoc, this.effectivePlanSoc || 0)
				);
				if (limit > this.vehicleSoc) {
					return limit - this.vehicleSoc;
				}
			} else {
				return 100 - this.vehicleSocDisplayWidth;
			}

			return null;
		},
		visibleLimitSoc: function () {
			return Number(this.selectedLimitSoc || this.effectiveLimitSoc);
		},
	},
	watch: {
		effectiveLimitSoc: function () {
			this.selectedLimitSoc = this.effectiveLimitSoc;
		},
		vehicleTargetSoc: function () {
			this.updateTooltip();
		},
	},
	mounted: function () {
		this.updateTooltip();
	},
	methods: {
		changeLimitSocStart: function (e) {
			this.dragging = true;
			e.stopPropagation();
		},
		changeLimitSocEnd: function (e) {
			this.dragging = false;
			const value = parseInt(e.target.value, 10);
			// value changed
			if (value !== this.effectiveLimitSoc) {
				this.$emit("limit-soc-updated", value);
			}
		},
		movedLimitSoc: function (e) {
			let value = parseInt(e.target.value, 10);
			e.stopPropagation();
			const minLimit = 20;
			if (value < minLimit) {
				e.target.value = minLimit;
				this.selectedLimitSoc = value;
				e.preventDefault();
				return false;
			}
			this.selectedLimitSoc = value;

			this.$emit("limit-soc-drag", this.selectedLimitSoc);
			return true;
		},
		updateTooltip: function () {
			if (!this.tooltip) {
				this.tooltip = new Tooltip(this.$refs.vehicleTargetSoc);
			}
			const soc = this.vehicleTargetSoc;
			const content = this.$t("main.vehicleSoc.vehicleTarget", { soc });
			this.tooltip.setContent({ ".tooltip-inner": content });
		},
	},
};
</script>
<style scoped>
.vehicle-soc {
	--height: 32px;
	--thumb-overlap: 6px;
	--thumb-width: 12px;
	--label-height: 26px;
	position: relative;
	height: var(--height);
}
.progress {
	height: 100%;
	font-size: 1rem;
	background: var(--evcc-background);
}
.progress-bar.bg-muted {
	opacity: 0.5;
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
.plan-marker--warning {
	color: var(--bs-warning);
}
.plan-marker--warning::before {
	background-color: var(--bs-warning);
}
.plan-marker--error {
	opacity: 1;
	color: var(--bs-danger);
}
.plan-marker--error::before {
	background-color: var(--bs-danger);
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
