<template>
	<div>
		<div class="mb-2">{{ socTitle || "Fahrzeug" }}</div>
		<div class="progress" style="height: 28px; font-size: 100%; margin-top: 16px">
			<div
				class="progress-bar"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
					'bg-light': !connected,
					'text-secondary': !connected,
					'bg-warning': connected && minSoCActive && !timerSet,
				}"
				:style="{ width: socChargeDisplayWidth + '%' }"
			>
				{{ socChargeDisplayValue }}
			</div>
			<div
				class="progress-bar"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
					'bg-muted': true,
				}"
				:style="{ width: targetSoCRemainingDisplayWidth + '%' }"
				v-if="timerSet && socChargeDisplayWidth < 100"
			></div>
			<div
				class="progress-bar"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
					'bg-warning': connected && minSoCActive,
					'bg-muted': true,
				}"
				:style="{ width: minSoCRemainingDisplayWidth + '%' }"
				v-else-if="minSoCActive && socChargeDisplayWidth < 100"
			></div>
		</div>
		<small
			v-if="connected && socMarker"
			:style="{
				paddingLeft: socMarker <= 50 ? `calc(${socMarker}% - 18px)` : null,
				paddingRight: socMarker > 50 ? `calc(${100 - socMarker}% - 18px)` : null,
			}"
			class="subline py-1"
			:class="{ 'subline--left': socMarker <= 50, 'subline--right': socMarker > 50 }"
		>
			<span class="subline__marker px-1">{{ socMarker }}%</span>
			<span class="text-muted">{{ markerLabel() }}</span>
			<fa-icon class="text-muted mx-1" :icon="timerSet ? 'clock' : 'first-aid'"></fa-icon>
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
		charging: Boolean,
		hasVehicle: Boolean,
		socCharge: Number,
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
			if (this.timerSet) {
				return this.targetSoC;
			}
			if (this.minSoCActive) {
				return this.minSoC;
			}
			return null;
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
		markerLabel: function () {
			if (this.timerSet) {
				const date = Date.parse(this.targetTime);
				return date ? this.fmtRelativeTime(date) + " geladen" : "";
			}
			if (this.minSoCActive) {
				return "Notladung";
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
.subline--right {
	text-align: right;
	flex-direction: row-reverse;
}
.progress {
	overflow: visible;
}
.progress-bar.bg-muted {
	position: relative;
	overflow: visible;
}
.progress-bar.bg-muted::after {
	position: absolute;
	right: 0;
	top: -3px;
	height: calc(100% + 6px);
	width: 1px;
	background: black;
	content: "";
}
</style>
