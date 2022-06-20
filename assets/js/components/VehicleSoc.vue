<template>
	<div class="vehicle-soc">
		<div class="progress">
			<div
				v-if="connected || parked"
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
		</div>
		<div class="target">
			<input
				v-if="vehiclePresent && (connected || parked)"
				type="range"
				min="0"
				max="100"
				step="5"
				:value="visibleTargetSoC"
				class="target-slider"
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
export default {
	name: "VehicleSoc",
	props: {
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		targetSoC: Number,
		parked: Boolean,
	},
	emits: ["target-soc-drag", "target-soc-updated"],
	data: function () {
		return {
			selectedTargetSoC: null,
			interactionStartScreenY: null,
		};
	},
	computed: {
		vehicleSoCDisplayWidth: function () {
			if (this.vehiclePresent && this.vehicleSoC >= 0) {
				return this.vehicleSoC;
			}
			return 100;
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
			if (this.vehicleSoCDisplayWidth === 100) {
				return null;
			}
			if (this.minSoCActive) {
				return this.minSoC - this.vehicleSoC;
			}
			if (this.visibleTargetSoC > this.vehicleSoC) {
				return this.visibleTargetSoC - this.vehicleSoC;
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
	background-color: var(--evcc-dark-green);
	cursor: grab;
	border: none;
	opacity: 1;
	border-radius: var(--thumb-overlap);
	box-shadow: 0 0 6px var(--evcc-background);
	pointer-events: auto;
}
.target-slider::-moz-range-thumb {
	position: relative;
	height: 100%;
	width: var(--thumb-width);
	background-color: var(--evcc-dark-green);
	cursor: grab;
	border: none;
	opacity: 1;
	border-radius: var(--thumb-overlap);
	box-shadow: 0 0 6px var(--evcc-background);
	pointer-events: auto;
}
</style>
