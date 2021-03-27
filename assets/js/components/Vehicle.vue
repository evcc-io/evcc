<template>
	<div>
		<div class="mb-2 pb-1">
			{{ socTitle || "Fahrzeug" }}
		</div>
		<div class="soc-bar">
			<div class="progress small">
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
					:style="{ width: `${remainingSoCWidth}%` }"
				></div>
				<div
					v-if="socMarker"
					class="soc-marker"
					:class="{ [progressColor]: true }"
					:style="{ left: `${socMarker}%` }"
				></div>
			</div>
		</div>
		<small v-if="markerLabel()" class="subline my-2 text-secondary">
			<fa-icon
				v-if="minSoCActive"
				class="text-muted mr-1"
				icon="exclamation-circle"
			></fa-icon>
			<fa-icon v-else-if="targetChargeEnabled" class="text-muted mr-1" icon="clock"></fa-icon>
			{{ markerLabel() }}
		</small>
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
			if (this.targetSoC > this.socCharge) {
				return this.targetSoC - this.socCharge;
			}
			return null;
		},
	},
	methods: {
		// not computed because it needs to update over time
		markerLabel: function () {
			if (!this.connected) {
				return null;
			}
			if (this.minSoCActive) {
				return `Mindestladung bis ${this.socMarker}%`;
			}
			if (this.targetChargeEnabled) {
				const targetDate = Date.parse(this.targetTime);
				if (this.timerActive) {
					return `Lädt ${this.fmtRelativeTime(targetDate)} bis ${this.socMarker}%`;
				} else {
					return `Geplant bis ${this.fmtAbsoluteDate(targetDate)} bis ${this.socMarker}%`;
				}
			}
			return null;
		},
	},
	mixins: [formatter],
};
</script>
<style scoped>
.subline {
	display: flex;
	align-items: center;
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
.soc-marker {
	position: absolute;
	top: -2px;
	bottom: -2px;
	width: 2px;
}
.bg-disabled {
	background-color: var(--gray);
}
.bg-light {
	color: var(--dark);
}
</style>
