<template>
	<div class="vehicle-soc">
		<div class="progress">
			<div
				class="progress-bar"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
					[progressColor]: true,
				}"
				:style="{ width: `${socChargeDisplayWidth}%` }"
			>
				{{ socChargeDisplayValue }}
			</div>
			<div
				v-if="remainingSoCWidth > 0"
				class="progress-bar"
				role="progressbar"
				:class="{
					[progressColor]: true,
					'bg-muted': true,
				}"
				:style="{ width: `${remainingSoCWidth}%`, transition: 'none' }"
			></div>
		</div>
		<div
			v-if="hasVehicle && visibleTargetSoC"
			class="target"
			:class="{ 'target--max': visibleTargetSoC === 100 }"
		>
			<div
				class="target-label d-flex align-items-center justify-content-center"
				:style="{ left: `${visibleTargetSoC}%` }"
			>
				{{ visibleTargetSoC }}%
			</div>
			<input
				type="range"
				min="0"
				max="100"
				step="5"
				:value="visibleTargetSoC"
				class="target-slider"
				@input="movedTargetSoC"
				@change="setTargetSoC"
			/>
		</div>
	</div>
</template>

<script>
export default {
	name: "VehicleSoc",
	props: {
		connected: Boolean,
		hasVehicle: Boolean,
		socCharge: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		targetSoC: Number,
	},
	data: function () {
		return {
			selectedTargetSoC: null,
		};
	},
	computed: {
		socChargeDisplayWidth: function () {
			if (this.hasVehicle && this.socCharge >= 0) {
				return this.socCharge;
			}
			return 100;
		},
		socChargeDisplayValue: function () {
			// no soc or no soc value
			if (!this.hasVehicle || !this.socCharge || this.socCharge < 0) {
				let chargeStatus = "getrennt";
				if (this.charging) {
					chargeStatus = "lädt";
				} else if (this.enabled) {
					chargeStatus = "bereit";
				} else if (this.connected) {
					chargeStatus = "verbunden";
				}
				return chargeStatus;
			}

			// percent value if enough space
			let socCharge = this.socCharge;
			if (socCharge >= 10) {
				socCharge += "%";
			}
			return socCharge;
		},
		progressColor: function () {
			if (!this.connected) {
				return "bg-light border";
			}
			if (this.minSoCActive) {
				return "bg-danger";
			}
			if (this.enabled) {
				return "bg-primary";
			}
			return "bg-secondary";
		},
		minSoCActive: function () {
			return this.minSoC > 0 && this.socCharge < this.minSoC;
		},
		remainingSoCWidth: function () {
			if (this.socCharge === 100) {
				return null;
			}
			if (this.minSoCActive) {
				return this.minSoC - this.socCharge;
			}
			if (this.visibleTargetSoC > this.socCharge) {
				return this.visibleTargetSoC - this.socCharge;
			}
			return null;
		},
		visibleTargetSoC: function () {
			return Number(this.selectedTargetSoC || this.targetSoC);
		},
	},
	methods: {
		movedTargetSoC: function (e) {
			const minTargetSoC = 40;
			if (e.target.value < minTargetSoC) {
				e.target.value = minTargetSoC;
				this.selectedTargetSoC = e.target.value;
				e.preventDefault();
				return false;
			}
			this.selectedTargetSoC = e.target.value;
			return true;
		},
		setTargetSoC: function (e) {
			this.$emit("target-soc-updated", e.target.value);
		},
	},
};
</script>
<style scoped>
.vehicle-soc {
	--height: 25px;
	--thumb-overlap: 3px;
	--thumb-width: 3px;
	--thumb-horizontal-padding: 15px;
	--label-height: 26px;
	position: relative;
	height: var(--height);
}
.progress {
	height: 100%;
	font-size: 0.875rem;
}
.progress-bar.bg-muted {
	color: var(--white);
}
.bg-light {
	color: var(--dark);
}
.target-label {
	width: 3em;
	margin-left: -1.5em;
	height: var(--label-height);
	position: absolute;
	top: calc((var(--label-height) + var(--thumb-overlap)) * -1);
	text-align: center;
	color: var(--dark);
	font-size: 0.875rem;
	opacity: 1;
	transition: opacity 0.2s ease 1s;
}
.target-slider {
	-webkit-appearance: none;
	position: absolute;
	top: calc(var(--thumb-overlap) * -1);
	left: calc(var(--thumb-horizontal-padding) * -1);
	height: calc(100% + 2 * var(--thumb-overlap));
	width: calc(100% + 2 * var(--thumb-horizontal-padding));
	background: transparent;
}
.target-slider:focus {
	outline: none;
}
/* Note: Safari,Chrome,Blink and Firefox specific styles need to be in separate definitions to work */
.target-slider::-webkit-slider-runnable-track {
	background: transparent;
	border: none;
	height: 100%;
	cursor: pointer;
}
.target-slider::-moz-range-track {
	background: transparent;
	border: none;
	height: 100%;
	cursor: pointer;
}
.target-slider::-webkit-slider-thumb {
	-webkit-appearance: none;
	position: relative;
	top: calc(var(--label-height) * -1);
	height: 100%;
	width: var(--thumb-width);
	padding: var(--label-height) var(--thumb-horizontal-padding) 0;
	box-sizing: content-box;
	background-clip: content-box;
	background-color: var(--dark);
	cursor: grab;
	border: none;
	opacity: 1;
	transition: opacity 0.2s ease 1s;
}
.target-slider::-moz-range-thumb {
	position: relative;
	top: calc(var(--label-height) * -1);
	height: 100%;
	width: var(--thumb-width);
	padding: var(--label-height) var(--thumb-horizontal-padding) 0;
	box-sizing: content-box;
	background-clip: content-box;
	background-color: var(--dark);
	cursor: grab;
	border: none;
	opacity: 1;
	transition: opacity 0.2s ease 1s;
}
/* auto-hide targetSoC marker at 100% */
.target--max .target-slider::-webkit-slider-thumb,
.target--max .target-label {
	opacity: 0;
}
.target--max .target-slider::-moz-range-thumb,
.target--max .target-label {
	opacity: 0;
}
.target:hover .target-slider::-webkit-slider-thumb,
.target:hover .target-label {
	opacity: 1;
	transition-delay: 0s;
}
.target:hover .target-slider::-moz-range-thumb,
.target:hover .target-label {
	opacity: 1;
	transition-delay: 0s;
}
</style>
