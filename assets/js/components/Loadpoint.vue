<template>
	<div class="border-top mt-4 pt-4">
		<div class="row">
			<div class="col-12">
				<h4>{{ title || "Ladepunkt" }}</h4>
			</div>

			<div class="col-12">
				<div class="row my-3 pb-3 bg-light">
					<div class="col-12 mt-3">
						<Mode
							:mode="mode"
							:pvConfigured="pvConfigured"
							v-on:updated="setTargetMode"
						></Mode>
					</div>
				</div>
			</div>
		</div>

		<div class="row">
			<div class="col-12 col-lg-5 pr-lg-4">
				<Vehicle
					v-bind="vehicle"
					@target-soc-updated="setTargetSoC"
					@target-time-updated="setTargetTime"
				></Vehicle>
			</div>

			<div class="col-12 col-lg-7 pl-lg-4">
				<LoadpointDetails v-bind="details"></LoadpointDetails>
			</div>
		</div>
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
		multi: Boolean,
		pvConfigured: Boolean,

		// main
		title: String,
		mode: String,
		targetSoC: Number,
		socLevels: Array,
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

		// details
		chargePower: Number,
		chargedEnergy: Number,
		// chargeDuration: Number,
		hasVehicle: Boolean,
		climater: String,
		range: Number,
		chargeEstimate: Number,
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
		hasTargetSoC: function () {
			return this.socLevels != null && this.socLevels.length > 0;
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
		setTargetTime: function (date) {
			const formattedDate = `${this.fmtDayString(date)}T${this.fmtTimeString(date)}:00`;
			axios
				.post(this.api("targetcharge") + "/" + this.targetSoC + "/" + formattedDate)
				.catch(window.toasts.error);
		},
	},
	destroyed: function () {
		window.clearInterval(this.tickerHandle);
	},
};
</script>
