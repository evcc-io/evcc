<template>
	<div>
		<div class="mb-2">{{ state.socTitle || "Fahrzeug" }}</div>
		<div class="progress" style="height: 24px; font-size: 100%; margin-top: 16px">
			<div
				class="progress-bar"
				role="progressbar"
				v-bind:class="{
					'progress-bar-striped': state.charging,
					'progress-bar-animated': state.charging,
					'bg-light': !state.connected,
					'text-secondary': !state.connected,
					'bg-warning': state.connected && minSoCActive,
				}"
				v-bind:style="{ width: socChargeDisplayWidth + '%' }"
			>
				{{ socChargeDisplayValue }}
			</div>
			<div
				class="progress-bar"
				role="progressbar"
				v-bind:class="{
					'progress-bar-striped': state.charging,
					'progress-bar-animated': state.charging,
					'bg-warning': true,
					'bg-muted': true,
				}"
				v-bind:style="{ width: minSoCRemainingDisplayWidth + '%' }"
				v-if="minSoCActive && socChargeDisplayWidth < 100"
			></div>
		</div>
	</div>
</template>

<script>
export default {
	name: "Vehicle",
	props: ["state"],
	computed: {
		socChargeDisplayWidth: function () {
			if (this.state.soc && this.state.socCharge >= 0) {
				return this.state.socCharge;
			}
			return 100;
		},
		socChargeDisplayValue: function () {
			// no soc or no soc value
			if (!this.state.soc || this.state.socCharge < 0) {
				let chargeStatus = "getrennt";
				if (this.state.charging) {
					chargeStatus = "laden";
				} else if (this.state.connected) {
					chargeStatus = "verbunden";
				}
				return chargeStatus;
			}

			// percent value if enough space
			let socCharge = this.state.socCharge;
			if (socCharge >= 10) {
				socCharge += "%";
			}
			return socCharge;
		},
		minSoCActive: function () {
			return this.state.minSoC > 0 && this.state.socCharge < this.state.minSoC;
		},
		minSoCRemainingDisplayWidth: function () {
			return this.state.minSoC - this.state.socCharge;
		},
	},
};
</script>
