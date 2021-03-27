<template>
	<div>
		<div class="mb-2 pb-1">
			{{ socTitle || "Fahrzeug" }}
		</div>
		<div class="soc-bar">
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
						'progress-bar-striped': charging,
						'progress-bar-animated': charging,
						[progressColor]: true,
						'bg-muted': true,
					}"
					:style="{ width: `${remainingSoCWidth}%`, transition: 'none' }"
				></div>
			</div>
			<div class="target-soc" v-if="hasVehicle && visibleTargetSoC">
				<div class="target-soc__label" :style="{ left: `${visibleTargetSoC}%` }">
					{{ visibleTargetSoC }}%
				</div>
				<input
					type="range"
					min="0"
					max="100"
					step="5"
					:value="visibleTargetSoC"
					class="target-soc__range"
					@input="movedTargetSoC"
					@change="setTargetSoC"
				/>
			</div>
		</div>
		<div class="subline my-1 text-secondary d-flex justify-content-between align-items-center">
			<div>
				<div v-if="minSoCActive">
					<fa-icon class="text-muted mr-1" icon="exclamation-circle"></fa-icon>
					{{ minChargeLabel() }}
				</div>
			</div>
			<button
				class="subline btn btn-link btn-sm pr-0"
				:class="{ 'text-dark': timerSet, 'text-muted': !timerSet }"
				@click="selectTargetTime"
			>
				<span>
					{{ targetTimeLabel() }}
				</span>
				<fa-icon class="ml-1" icon="clock"></fa-icon>
			</button>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "Vehicle",
	props: {
		socTitle: String,
		connected: Boolean,
		hasVehicle: Boolean,
		socCharge: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		timerActive: Boolean,
		timerSet: Boolean,
		targetTime: String,
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
		socMarker: function () {
			if (this.minSoCActive) {
				return this.minSoC;
			}
			if (this.targetSoC > this.socCharge) {
				return this.targetSoC;
			}
			return null;
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
		targetChargeEnabled: function () {
			return this.targetTime && this.timerSet;
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
			return this.selectedTargetSoC || this.targetSoC;
		},
	},
	methods: {
		// not computed because it needs to update over time
		minChargeLabel: function () {
			if (this.connected && this.minSoCActive) {
				return `Mindestladung bis ${this.socMarker}%`;
			}
			return null;
		},
		targetTimeLabel: function () {
			if (this.targetChargeEnabled) {
				const targetDate = Date.parse(this.targetTime);
				if (this.timerActive) {
					return `Lädt ${this.fmtRelativeTime(targetDate)} bis ${this.socMarker}%`;
				} else {
					return `Geplant bis ${this.fmtAbsoluteDate(targetDate)} bis ${this.socMarker}%`;
				}
			}
			return "Zielzeit festlegen";
		},
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
			const { value } = e.target;
			this.$emit("target-soc-updated", value);
		},
		selectTargetTime: function () {
			window.alert("Bis wann soll geladen werden?");
		},
	},
	mixins: [formatter],
};
</script>
<style scoped>
.subline {
	display: flex;
	align-items: center;
	font-size: 0.875rem;
}
.soc-bar {
	position: relative;
	height: 31px;
}
.progress {
	height: 100%;
	font-size: 0.875rem;
}
.progress-bar.bg-muted {
	color: var(--white);
}
.target-soc__label {
	width: 3em;
	margin-left: -1.5em;
	position: absolute;
	top: -90%;
	text-align: center;
	color: var(--dark);
	font-size: 0.875rem;
}
.target-soc__range {
	-webkit-appearance: none;
	position: absolute;
	top: 0;
	left: -15px;
	height: 100%;
	width: calc(100% + 2 * 15px);
	background: transparent;
}
.target-soc__range:focus {
	outline: none;
}
.target-soc__range::-webkit-slider-runnable-track {
	background: transparent;
	border: none;
	height: 100%;
	cursor: pointer;
}
.target-soc__range::-moz-range-track {
	background: transparent;
	border: none;
	height: 100%;
	cursor: pointer;
}
.target-soc__range::-webkit-slider-thumb {
	-webkit-appearance: none;
	height: 100%;
	width: 3px;
	padding: 0 15px;
	box-sizing: content-box;
	background-clip: content-box;
	background-color: var(--dark);
	cursor: grab;
	border: none;
	transform: scaleY(1.2);
}
.target-soc__range::-moz-range-thumb {
	height: 100%;
	width: 3px;
	padding: 0 15px;
	box-sizing: content-box;
	background-clip: content-box;
	background-color: var(--primary);
	cursor: grab;
	border: none;
	transform: scaleY(1.2);
}
.bg-disabled {
	background-color: var(--gray);
}
.bg-light {
	color: var(--dark);
}
</style>
