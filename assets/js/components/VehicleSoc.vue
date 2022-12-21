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
				:style="{ width: `${vehicleSoCDisplayWidth}%` }"
			></div>
			<div
				v-if="remainingSoCWidth > 0 && enabled && connected"
				class="progress-bar bg-muted"
				role="progressbar"
				:class="progressColor"
				:style="{ width: `${remainingSoCWidth}%`, transition: 'none' }"
			></div>
			<div
				v-show="vehicleTargetSoC"
				ref="vehicleTargetSoC"
				class="vehicle-target-soc"
				data-bs-toggle="tooltip"
				title=" "
				:class="{ 'vehicle-target-soc--active': vehicleTargetSoCActive }"
				:style="{ left: `${vehicleTargetSoC}%` }"
			/>
		</div>
		<div class="target">
			<input
				v-if="socBasedCharging && connected"
				type="range"
				min="0"
				max="100"
				step="5"
				:value="visibleTargetSoC"
				class="target-slider"
				:class="{ 'target-slider--active': targetSliderActive }"
				@mousedown="changeTargetSoCStart"
				@touchstart="changeTargetSoCStart"
				@input="movedTargetSoC"
				@mouseup="changeTargetSoCEnd"
				@touchend="changeTargetSoCEnd"
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
		vehicleSoC: Number,
		vehicleTargetSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		targetSoC: Number,
		targetEnergy: Number,
		chargedEnergy: Number,
		socBasedCharging: Boolean,
	},
	emits: ["target-soc-drag", "target-soc-updated"],
	data: function () {
		return {
			selectedTargetSoC: null,
			interactionStartScreenY: null,
			tooltip: null,
		};
	},
	computed: {
		vehicleSoCDisplayWidth: function () {
			if (this.socBasedCharging) {
				if (this.vehicleSoC >= 0) {
					return this.vehicleSoC;
				}
				return 100;
			} else {
				if (this.targetEnergy) {
					return (100 / this.targetEnergy) * (this.chargedEnergy / 1e3);
				}
				return 100;
			}
		},
		vehicleTargetSoCActive: function () {
			return this.vehicleTargetSoC > 0 && this.vehicleTargetSoC > this.vehicleSoC;
		},
		targetSliderActive: function () {
			return !this.vehicleTargetSoC || this.visibleTargetSoC <= this.vehicleTargetSoC;
		},
		progressColor: function () {
			if (this.minSoCActive) {
				return "bg-danger";
			}
			return "bg-primary";
		},
		minSoCActive: function () {
			return this.minSoC > 0 && this.vehicleSoC < this.minSoC;
		},
		remainingSoCWidth: function () {
			if (this.socBasedCharging) {
				if (this.vehicleSoCDisplayWidth === 100) {
					return null;
				}
				if (this.minSoCActive) {
					return this.minSoC - this.vehicleSoC;
				}
				let targetSoC = this.targetSliderActive
					? this.visibleTargetSoC
					: this.vehicleTargetSoC;
				if (targetSoC > this.vehicleSoC) {
					return targetSoC - this.vehicleSoC;
				}
			} else {
				return 100 - this.vehicleSoCDisplayWidth;
			}

			return null;
		},
		visibleTargetSoC: function () {
			return Number(this.selectedTargetSoC || this.targetSoC);
		},
	},
	watch: {
		targetSoC: function () {
			this.selectedTargetSoC = this.targetSoC;
		},
		vehicleTargetSoC: function () {
			this.updateTooltip();
		},
	},
	mounted: function () {
		this.updateTooltip();
	},
	methods: {
		changeTargetSoCStart: function (e) {
			e.stopPropagation();
		},
		changeTargetSoCEnd: function (e) {
			const value = parseInt(e.target.value, 10);
			// value changed
			if (value !== this.targetSoC) {
				this.$emit("target-soc-updated", value);
			}
		},
		movedTargetSoC: function (e) {
			let value = parseInt(e.target.value, 10);
			e.stopPropagation();
			const minTargetSoC = 20;
			if (value < minTargetSoC) {
				e.target.value = minTargetSoC;
				this.selectedTargetSoC = value;
				e.preventDefault();
				return false;
			}
			this.selectedTargetSoC = value;

			this.$emit("target-soc-drag", this.selectedTargetSoC);
			return true;
		},
		updateTooltip: function () {
			if (!this.tooltip) {
				this.tooltip = new Tooltip(this.$refs.vehicleTargetSoC);
			}
			const soc = this.vehicleTargetSoC;
			const content = this.$t("main.vehicleSoC.vehicleTarget", { soc });
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
.target-slider {
	-webkit-appearance: none;
	position: absolute;
	top: calc(var(--thumb-overlap) * -1);
	height: calc(100% + 2 * var(--thumb-overlap));
	width: 100%;
	background: transparent;
	pointer-events: none;
}
.target-slider:focus {
	outline: none;
}
/* Note: Safari,Chrome,Blink and Firefox specific styles need to be in separate definitions to work */
.target-slider::-webkit-slider-runnable-track {
	position: relative;
	background: transparent;
	border: none;
	height: 100%;
	cursor: auto;
}
.target-slider::-moz-range-track {
	background: transparent;
	border: none;
	height: 100%;
	cursor: auto;
}
.target-slider::-webkit-slider-thumb {
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
.target-slider::-moz-range-thumb {
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
.target-slider--active::-webkit-slider-thumb {
	background-color: var(--evcc-dark-green);
}
.target-slider--active::-moz-range-thumb {
	background-color: var(--evcc-dark-green);
}
.vehicle-target-soc {
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
.vehicle-target-soc--active {
	background-color: var(--evcc-box);
}
</style>
