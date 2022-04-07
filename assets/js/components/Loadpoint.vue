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
				@target-time-removed="removeTargetTime"
			/>
		</div>
		<LoadpointDetails v-bind="details" />
	</div>
</template>

<script>
import api from "../api";
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
		targetTime: String,
		targetTimeActive: Boolean,
		vehicleProviderLoggedIn: Boolean,
		vehicleProviderLoginPath: String,
		vehicleProviderLogoutPath: String,

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
	computed: {
		details: function () {
			return this.collectProps(LoadpointDetails);
		},
		vehicle: function () {
			return this.collectProps(Vehicle);
		},
	},
	methods: {
		apiPath: function (func) {
			return "loadpoints/" + this.id + "/" + func;
		},
		setTargetMode: function (mode) {
			api.post(this.apiPath("mode") + "/" + mode);
		},
		setTargetSoC: function (soc) {
			api.post(this.apiPath("targetsoc") + "/" + soc);
		},
		setTargetTime: function (date) {
			const formattedDate = `${this.fmtDayString(date)}T${this.fmtTimeString(date)}:00`;
			api.post(this.apiPath("targetcharge") + "/" + this.targetSoC + "/" + formattedDate);
		},
		removeTargetTime: function () {
			api.delete(this.apiPath("targetcharge"));
		},
	},
};
</script>
