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
		<small
			v-if="connected && socMarker"
			:style="{
				paddingLeft: socMarker <= 50 ? `calc(${socMarker}% - 14px)` : null,
				paddingRight: socMarker > 50 ? `calc(${100 - socMarker}% - 22px)` : null,
			}"
			class="subline py-1"
			:class="{ 'subline--left': socMarker <= 50, 'subline--right': socMarker > 50 }"
		>
			<span class="subline__marker px-1">{{ socMarker }}%</span>
			<span class="text-muted">{{ markerLabel() }}</span>
			<fa-icon class="text-muted mx-1" :icon="minSoCActive ? 'first-aid' : 'clock'"></fa-icon>
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
				return "bg-success";
			}
			if (!this.connected) {
				return "bg-light";
			}
			return "bg-disabled";
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
				return "Notladung";
			}
			if (this.timerActive) {
				const date = Date.parse(this.targetTime);
				return date ? this.fmtRelativeTime(date) + " geladen" : "";
			}
			if (this.timerSet) {
				const date = Date.parse(this.targetTime);
				return date ? "bis " + this.fmtAbsoluteDate(date) : "";
			}
		},
	},
	mixins: [formatter],
};
</script>
<style scoped>
.subline {
	display: flex;
	transition: padding 0.6s ease;
	align-items: center;
}
.subline--left {
	justify-content: flex-start;
}
.subline--right {
	flex-direction: row-reverse;
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
	top: -3px;
	height: calc(100% + 6px);
	width: 1px;
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
