<template>
	<div>
		<p class="h3 mb-4 d-sm-block" :class="{ 'd-none': single }">
			{{ title || $t("main.loadpoint.fallbackName") }}
		</p>
		<div v-if="remoteDisabled == 'soft'" class="alert alert-warning mt-4 mb-2" role="alert">
			{{ $t("main.loadpoint.remoteDisabledSoft", { source: remoteDisabledSource }) }}
		</div>
		<div v-if="remoteDisabled == 'hard'" class="alert alert-danger mt-4 mb-2" role="alert">
			{{ $t("main.loadpoint.remoteDisabledHard", { source: remoteDisabledSource }) }}
		</div>

		<div class="row">
			<Mode class="col-12 col-md-6 col-lg-4 mb-4" :mode="mode" @updated="setTargetMode" />
			<Vehicle
				class="col-12 col-md-6 col-lg-8 mb-4"
				v-bind="vehicle"
				@target-soc-updated="setTargetSoC"
				@target-time-updated="setTargetTime"
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
	components: { LoadpointDetails, Mode, Vehicle },
	mixins: [formatter, collector],
	props: {
		id: Number,
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
		vehicleTitle: String,
		vehicleSoC: Number,
		vehiclePresent: Boolean,
		vehicleRange: Number,
		minSoC: Number,
		timerSet: Boolean,
		timerActive: Boolean,
		targetTime: String,

		// details
		chargePower: Number,
		chargedEnergy: Number,
		// chargeDuration: Number,
		climater: String,
		chargeRemainingDuration: Number,

		// other information
		phases: Number,
		minCurrent: Number,
		maxCurrent: Number,
		activePhases: Number,
		chargeCurrent: Number,
		vehicleCapacity: Number,
		connectedDuration: Number,
		chargeCurrents: Array,
		chargeConfigured: Boolean,
		chargeRemainingEnergy: Number,
		phaseAction: String,
		phaseRemaining: Number,
		pvRemaining: Number,
		pvAction: String,
	},
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
	destroyed: function () {
		window.clearInterval(this.tickerHandle);
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
				.catch(window.app.error);
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
				.catch(window.app.error);
		},
		setTargetTime: function (date) {
			const formattedDate = `${this.fmtDayString(date)}T${this.fmtTimeString(date)}:00`;
			axios
				.post(this.api("targetcharge") + "/" + this.targetSoC + "/" + formattedDate)
				.catch(window.app.error);
		},
	},
};
</script>
