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
				:style="{ width: `${vehicleSoCDisplayWidth}%` }"
			>
				{{ vehicleSoCDisplayValue }}
			</div>
			<div
				v-if="remainingSoCWidth > 0 && enabled"
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
			class="target"
			:class="{ 'target--slider-hidden': allowSliderHiding && visibleTargetSoC === 100 }"
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
				@mousedown="changeTargetSoCStart"
				@touchstart="changeTargetSoCStart"
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
	},
	data: function () {
		return {
			selectedTargetSoC: null,
			allowSliderHiding: false,
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
		vehicleSoCDisplayValue: function () {
			// no soc or no soc value
			if (!this.vehiclePresent || !this.vehicleSoC || this.vehicleSoC < 0) {
				let chargeStatus = this.$t("main.vehicleSoC.disconnected");
				if (this.charging) {
					chargeStatus = this.$t("main.vehicleSoC.charging");
				} else if (this.enabled) {
					chargeStatus = this.$t("main.vehicleSoC.ready");
				} else if (this.connected) {
					chargeStatus = this.$t("main.vehicleSoC.connected");
				}
				return chargeStatus;
			}

			// percent value if enough space
			let vehicleSoC = this.vehicleSoC;
			if (vehicleSoC >= 10) {
				vehicleSoC += "%";
			}
			return vehicleSoC;
		},
		progressColor: function () {
			if (!this.connected) {
				return "bg-light border";
			}
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
	mounted: function () {
		setTimeout(() => {
			this.allowSliderHiding = true;
		}, 1000);
	},
	methods: {
		changeTargetSoCStart: function (e) {
			const screenY = e.screenY || e.changedTouches[0].screenY;
			this.interactionStartScreenY = screenY;
		},
		changeTargetSoCEnd: function (e) {
			const screenY = e.screenY || e.changedTouches[0].screenY;
			const yDiff = Math.abs(screenY - this.interactionStartScreenY);
			// horizontal scroll detected - revert slider change
			if (yDiff > 80) {
				e.preventDefault();
				e.target.value = this.targetSoC;
				this.selectedTargetSoC = this.targetSoC;
				return false;
			}
			// value changed
			if (e.target.value !== this.targetSoC) {
				this.$emit("target-soc-updated", e.target.value);
			}
		},
		movedTargetSoC: function (e) {
			const minTargetSoC = 20;
			if (e.target.value < minTargetSoC) {
				e.target.value = minTargetSoC;
				this.selectedTargetSoC = e.target.value;
				e.preventDefault();
				return false;
			}
			this.selectedTargetSoC = e.target.value;
			return true;
		},
	},
};
</script>
<style scoped>
.vehicle-soc {
	--height: 38px;
	--thumb-overlap: 3px;
	--thumb-width: 3px;
	--thumb-horizontal-padding: 15px;
	--label-height: 26px;
	position: relative;
	height: var(--height);
}
.progress {
	height: 100%;
	font-size: 1rem;
}
.progress-bar.bg-muted {
	color: var(--white);
}
.bg-light {
	color: var(--bs-gray-dark);
}
.target-label {
	width: 3em;
	margin-left: -1.5em;
	height: var(--label-height);
	position: absolute;
	top: calc((var(--label-height) + var(--thumb-overlap)) * -1);
	text-align: center;
	color: var(--bs-gray-dark);
	font-size: 1rem;
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
	position: relative;
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
	background-color: var(--bs-gray-dark);
	cursor: grab;
	border: none;
	opacity: 1;
	transition: opacity 0.2s ease 1s;
	box-shadow: none;
}
.target-slider::-moz-range-thumb {
	position: relative;
	top: calc(var(--label-height) * -1);
	height: 100%;
	width: var(--thumb-width);
	padding: 0 var(--thumb-horizontal-padding) 0;
	box-sizing: content-box;
	background-clip: content-box;
	background-color: var(--bs-gray-dark);
	cursor: grab;
	border: none;
	opacity: 1;
	transition: opacity 0.5s ease 1s;
}
/* auto-hide targetSoC marker at 100% */
.target--slider-hidden .target-slider::-webkit-slider-thumb,
.target--slider-hidden .target-label {
	opacity: 0;
}
.target--slider-hidden .target-slider::-moz-range-thumb,
.target--slider-hidden .target-label {
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
