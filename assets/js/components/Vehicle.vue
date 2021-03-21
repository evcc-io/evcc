<template>
	<div>
		<div class="mb-3">
			{{ socTitle || "Fahrzeug" }}
		</div>
		<div class="progress" style="height: 28px; font-size: 100%">
			<div
				class="progress-bar"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
					[progressColor]: true,
				}"
				:style="{ width: socChargeDisplayWidth + '%' }"
			>
				{{ socChargeDisplayValue }}
			</div>
			<div
				v-if="minSoCActive && socChargeDisplayWidth < 100"
				class="progress-bar"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
					[progressColor]: true,
					'bg-muted': true,
				}"
				:style="{ width: minSoCRemainingDisplayWidth + '%' }"
			></div>
			<div
				v-else-if="timerSet && socChargeDisplayWidth < 100"
				class="progress-bar"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
					[progressColor]: true,
					'bg-muted': true,
				}"
				:style="{ width: targetSoCRemainingDisplayWidth + '%' }"
			></div>
		</div>
		<small v-if="hasVehicle && markerLabel()" class="subline py-2">
			<fa-icon v-if="minSoCActive" class="text-muted mr-1" icon="first-aid"></fa-icon>
			<fa-icon v-else-if="timerSet" class="text-muted mr-1" icon="clock"></fa-icon>
			<fa-icon v-else class="text-muted mr-1" icon="bolt"></fa-icon>
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
			if (!this.hasVehicle || this.socCharge < 0) {
				let chargeStatus = "getrennt";
				if (this.charging) {
					chargeStatus = "laden";
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
			if (!this.connected || !this.hasVehicle) {
				return null;
			}
			if (this.minSoCActive) {
				return this.minSoC;
			}
			if (this.timerSet) {
				return this.targetSoC;
			}
			return null;
		},
		progressColor: function () {
			if (this.minSoCActive) {
				return "bg-danger";
			}
			if (this.enabled) {
				return "bg-primary";
			}
			if (!this.connected) {
				return "bg-light";
			}
			return "bg-secondary";
		},
		minSoCActive: function () {
			return this.minSoC > 0 && this.socCharge < this.minSoC;
		},
		targetSoCRemainingDisplayWidth: function () {
			return this.targetSoC - this.socCharge;
		},
		minSoCRemainingDisplayWidth: function () {
			return this.minSoC - this.socCharge;
		},
	},
	methods: {
		// not computed because it needs to update over time
		markerLabel: function () {
			if (this.minSoCActive) {
				return `Notladung bis ${this.socMarker}%`;
			}
			if (this.timerActive) {
				const date = Date.parse(this.targetTime);
				return date ? `Lädt ${this.fmtRelativeTime(date)} bis ${this.socMarker}%` : null;
			}
			if (this.timerSet) {
				const date = Date.parse(this.targetTime);
				return date
					? `Geplant bis ${this.fmtAbsoluteDate(date)} bis ${this.socMarker}%`
					: null;
			}
			if (this.enabled && !this.charging) {
				return "Warte auf Fahrzeug";
			}
			if (!this.enabled) {
				return "Ladung noch nicht freigegeben";
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
.progress {
	overflow: visible;
}
.progress-bar.bg-muted {
	position: relative;
	overflow: visible;
	color: var(--white);
}
.progress-bar.bg-muted::after {
	position: absolute;
	right: 0;
	top: -5px;
	height: calc(100% + 10px);
	width: 2px;
	background: var(--dark);
	content: "";
}
.bg-disabled {
	background-color: var(--gray);
}
.bg-light {
	color: var(--gray);
}
</style>
