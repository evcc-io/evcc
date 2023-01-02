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
				class="vehicle-target-soc"
				data-bs-toggle="tooltip"
				title=" "
				:class="{ 'vehicle-target-soc--active': vehicleTargetSocActive }"
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
				:value="visibleTargetSoc"
				class="target-slider"
				:class="{ 'target-slider--active': targetSliderActive }"
				@mousedown="changeTargetSocStart"
				@touchstart="changeTargetSocStart"
				@input="movedTargetSoc"
				@mouseup="changeTargetSocEnd"
				@touchend="changeTargetSocEnd"
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
		targetSoc: Number,
		targetEnergy: Number,
		chargedEnergy: Number,
		socBasedCharging: Boolean,
	},
	emits: ["target-soc-drag", "target-soc-updated"],
	data: function () {
		return {
			selectedTargetSoc: null,
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
				if (this.targetEnergy) {
					return (100 / this.targetEnergy) * (this.chargedEnergy / 1e3);
				}
				return 100;
			}
		},
		vehicleTargetSocActive: function () {
			return this.vehicleTargetSoc > 0 && this.vehicleTargetSoc > this.vehicleSoc;
		},
		targetSliderActive: function () {
			return !this.vehicleTargetSoc || this.visibleTargetSoc <= this.vehicleTargetSoc;
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
				let targetSoc = this.targetSliderActive
					? this.visibleTargetSoc
					: this.vehicleTargetSoc;
				if (targetSoc > this.vehicleSoc) {
					return targetSoc - this.vehicleSoc;
				}
			} else {
				return 100 - this.vehicleSocDisplayWidth;
			}

			return null;
		},
		visibleTargetSoc: function () {
			return Number(this.selectedTargetSoc || this.targetSoc);
		},
	},
	watch: {
		targetSoc: function () {
			this.selectedTargetSoc = this.targetSoc;
		},
		vehicleTargetSoc: function () {
			this.updateTooltip();
		},
	},
	mounted: function () {
		this.updateTooltip();
	},
	methods: {
		changeTargetSocStart: function (e) {
			e.stopPropagation();
		},
		changeTargetSocEnd: function (e) {
			const value = parseInt(e.target.value, 10);
			// value changed
			if (value !== this.targetSoc) {
				this.$emit("target-soc-updated", value);
			}
		},
		movedTargetSoc: function (e) {
			let value = parseInt(e.target.value, 10);
			e.stopPropagation();
			const minTargetSoc = 20;
			if (value < minTargetSoc) {
				e.target.value = minTargetSoc;
				this.selectedTargetSoc = value;
				e.preventDefault();
				return false;
			}
			this.selectedTargetSoc = value;

			this.$emit("target-soc-drag", this.selectedTargetSoc);
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
