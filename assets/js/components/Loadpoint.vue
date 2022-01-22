<template>
	<div class="loadpoint bg-white px-4 px-sm-5 py-4 mb-3 mb-sm-4">
		<div class="d-flex justify-content-between align-items-center mb-4 flex-wrap">
			<h3 class="mb-2 me-2">
				{{ title || $t("main.loadpoint.fallbackName") }}
			</h3>
			<Mode class="mb-2" :mode="mode" @updated="setTargetMode" />
		</div>
		<div v-if="remoteDisabled == 'soft'" class="alert alert-warning mt-4 mb-2" role="alert">
			{{ $t("main.loadpoint.remoteDisabledSoft", { source: remoteDisabledSource }) }}
		</div>
		<div v-if="remoteDisabled == 'hard'" class="alert alert-danger mt-4 mb-2" role="alert">
			{{ $t("main.loadpoint.remoteDisabledHard", { source: remoteDisabledSource }) }}
		</div>

		<LoadpointDetails v-bind="details" />
		<Vehicle
			v-bind="vehicle"
			@target-soc-updated="setTargetSoC"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
		/>
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
		targetTimeHourSuggestion: Number,

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

<style scoped>
.loadpoint {
	border-radius: 20px;
	color: var(--bs-gray-dark);
}
</style>
