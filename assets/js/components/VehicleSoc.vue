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
				:style="{ width: `${vehicleSocDisplayWidth}%` }"
			></div>
			<div
				v-if="remainingSocWidth > 0 && enabled && connected"
				class="progress-bar bg-muted"
				role="progressbar"
				:class="progressColor"
				:style="{ width: `${remainingSocWidth}%`, transition: 'none' }"
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
		</div>
		<div class="target">
			<input
				v-if="socBasedCharging && connected"
				type="range"
				min="0"
				max="100"
				step="5"
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

export default {
	name: "VehicleSoc",
	props: {
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoc: Number,
		vehicleTargetSoc: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoc: Number,
		effectiveLimitSoc: Number,
		limitEnergy: Number,
		chargedEnergy: Number,
		socBasedCharging: Boolean,
	},
	emits: ["limit-soc-drag", "limit-soc-updated"],
	data: function () {
		return {
			selectedLimitSoc: null,
			interactionStartScreenY: null,
			tooltip: null,
		};
	},
	computed: {
		vehicleSocDisplayWidth: function () {
			if (this.socBasedCharging) {
				if (this.vehicleSoc >= 0) {
					return this.vehicleSoc;
				}
				return 100;
			} else {
				if (this.limitEnergy) {
					return (100 / this.limitEnergy) * (this.chargedEnergy / 1e3);
				}
				return 100;
			}
		},
		vehicleTargetSocActive: function () {
			return this.vehicleTargetSoc > 0 && this.vehicleTargetSoc > this.vehicleSoc;
		},
		sliderActive: function () {
			return !this.vehicleTargetSoc || this.visibleLimitSoc <= this.vehicleTargetSoc;
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
				let soc = this.sliderActive ? this.visibleLimitSoc : this.vehicleTargetSoc;
				if (soc > this.vehicleSoc) {
					return soc - this.vehicleSoc;
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
			e.stopPropagation();
		},
		changeLimitSocEnd: function (e) {
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
}
.slider::-moz-range-track {
	background: transparent;
	border: none;
	height: 100%;
	cursor: auto;
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
	transition: background-color var(--evcc-transition-fast) linear;
}
.vehicle-limit-soc--active {
	background-color: var(--evcc-box);
}
</style>
