<template>
	<div>
		<p class="h3 mb-4 d-sm-block" :class="{ 'd-none': single }">{{ title || "Ladepunkt" }}</p>
		<div class="alert alert-warning mt-4 mb-2" role="alert" v-if="remoteDisabled == 'soft'">
			{{ remoteDisabledSource }}: Adaptives PV-Laden deaktiviert
		</div>
		<div class="alert alert-danger mt-4 mb-2" role="alert" v-if="remoteDisabled == 'hard'">
			{{ remoteDisabledSource }}: Deaktiviert
		</div>

		<div class="row">
			<Mode
				class="col-12 col-md-6 col-lg-4 mb-4"
				:mode="mode"
				:pvConfigured="pvConfigured"
				v-on:updated="setTargetMode"
			/>
			<Vehicle
				class="col-12 col-md-6 col-lg-8 mb-4"
				v-bind="vehicle"
				@target-soc-updated="setTargetSoC"
			/>
		</div>
		<LoadpointDetails v-bind="details" />
	</div>
</template>

<script>
import axios from "axios";
import Mode from "./Mode";
import Vehicle from "./Vehicle";
import LoadpointDetails from "./LoadpointDetails";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";

export default {
	name: "Loadpoint",
	props: {
		id: Number,
		pvConfigured: Boolean,
		single: Boolean,

		// main
		title: String,
		mode: String,
		targetSoC: Number,
		remoteDisabled: Boolean,
		remoteDisabledSource: String,
		chargeDuration: Number,
		charging: Boolean,

		// vehicle
		connected: Boolean,
		// charging: Boolean,
		enabled: Boolean,
		socTitle: String,
		socCharge: Number,
		minSoC: Number,
		timerSet: Boolean,
		timerActive: Boolean,
		targetTime: String,
		vehicleProviderLoggedIn: Boolean,
		vehicleProviderLoginPath: String,

		// details
		chargePower: Number,
		chargedEnergy: Number,
		// chargeDuration: Number,
		hasVehicle: Boolean,
		climater: String,
		range: Number,
		chargeEstimate: Number,

		// other information
		phases: Number,
		minCurrent: Number,
		maxCurrent: Number,
		activePhases: Number,
		chargeCurrent: Number,
		socCapacity: Number,
		connectedDuration: Number,
		chargeCurrents: Object,
		chargeConfigured: Boolean,
		chargeRemainingEnergy: Number,
	},
	components: { LoadpointDetails, Mode, Vehicle },
	mixins: [formatter, collector],
	data: function () {
		return {
			tickerHandle: null,
			chargeDurationDisplayed: null,
		};
	},
	computed: {
		details: function () {
			return this.collectProps(LoadpointDetails);
		},
		vehicle: function () {
			return this.collectProps(Vehicle);
		},
	},
	watch: {
		chargeDuration: function () {
			window.clearInterval(this.tickerHandle);
			// only ticker if actually charging
			if (this.charging && this.chargeDuration >= 0) {
				this.chargeDurationDisplayed = this.chargeDuration;
				this.tickerHandle = window.setInterval(
					function () {
						this.chargeDurationDisplayed += 1;
					}.bind(this),
					1000
				);
			}
		},
	},
	methods: {
		api: function (func) {
			return "loadpoints/" + this.id + "/" + func;
		},
		setTargetMode: function (mode) {
			axios
				.post(this.api("mode") + "/" + mode)
				.then(
					function (response) {
						// eslint-disable-next-line vue/no-mutating-props
						this.mode = response.data.mode;
					}.bind(this)
				)
				.catch(window.toasts.error);
		},
		setTargetSoC: function (soc) {
			axios
				.post(this.api("targetsoc") + "/" + soc)
				.then(
					function (response) {
						// eslint-disable-next-line vue/no-mutating-props
						this.targetSoC = response.data.targetSoC;
					}.bind(this)
				)
				.catch(window.toasts.error);
		},
	},
	destroyed: function () {
		window.clearInterval(this.tickerHandle);
	},
};
</script>
