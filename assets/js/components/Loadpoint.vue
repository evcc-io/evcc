<template>
	<div>
		<div class="row" v-if="multi">
			<div class="col-12 col-md-4 d-md-flex mt-3 mt-md-5 align-items-end">
				<span class="h1 align-bottom">{{ title || "Ladepunkt" }}</span>
			</div>

			<div class="col-12 col-md-8 d-none d-md-block mt-3 mt-md-5">
				<LoadpointDetails v-bind="details"> </LoadpointDetails>
			</div>

			<div class="col-12 d-md-none">
				<div class="row mt-3 pb-3 bg-light">
					<div class="col-12 mt-3">
						<Mode
							class="w-100"
							:mode="mode"
							:pvConfigured="pvConfigured"
							v-on:updated="setTargetMode"
						></Mode>
					</div>
					<div class="col-12 mt-3" v-if="hasTargetSoC">
						<Soc
							class="w-100"
							:soc="targetSoC"
							:levels="socLevels"
							v-on:updated="setTargetSoC"
						></Soc>
					</div>
				</div>
			</div>
		</div>

		<div class="row d-none d-md-flex mt-5 py-3 pb-4 text-center bg-light" v-if="!multi">
			<div class="mt-3" :class="{ 'col-md-6': hasTargetSoC, 'col-md-12': !hasTargetSoC }">
				<Mode
					:mode="mode"
					:pvConfigured="pvConfigured"
					:caption="true"
					v-on:updated="setTargetMode"
				></Mode>
			</div>
			<div class="col-md-6 mt-3" v-if="hasTargetSoC">
				<Soc
					:soc="targetSoC"
					:levels="socLevels"
					:caption="true"
					v-on:updated="setTargetSoC"
				></Soc>
				<!-- <div class="btn-group btn-group-toggle bg-white shadow-none">
					<label class="btn btn-outline-primary">
						<input
							type="checkbox"
							class="disabled"
							v-on:click="alert('not implemented - use api')"
						/>
						<fa-icon
							icon="clock"
							v-bind:class="{ fas: timerActive, far: !timerActive }"
						></fa-icon>
					</label>
				</div> -->
			</div>
		</div>

		<div class="row d-md-none mt-2 pb-3 bg-light" v-if="!multi">
			<div class="col-12 mt-3">
				<Mode
					class="w-100"
					:mode="mode"
					:pvConfigured="pvConfigured"
					v-on:updated="setTargetMode"
				></Mode>
			</div>
			<div class="col-12 mt-3" v-if="hasTargetSoC">
				<Soc
					class="w-100"
					:soc="targetSoC"
					:levels="socLevels"
					v-on:updated="setTargetSoC"
				></Soc>
			</div>
		</div>

		<div class="row" v-if="!multi">
			<div class="col-12 col-md-4 d-none d-md-flex mt-3 mt-md-5">
				<span class="h1">{{ title || "Ladepunkt" }}</span>
			</div>
			<div class="col-12 col-md-8 d-flex d-md-flex mt-3 mt-md-5 pt-3" v-if="remoteDisabled">
				<h5 class="w-100">
					<span class="badge badge-warning w-100" v-if="remoteDisabled == 'soft'">
						{{ remoteDisabledSource }}: Adaptives PV-Laden deaktiviert
					</span>
					<span class="badge badge-danger w-100" v-if="remoteDisabled == 'hard'">
						{{ remoteDisabledSource }}: Deaktiviert
					</span>
				</h5>
			</div>
		</div>

		<div class="row border-bottom d-none d-md-block"></div>

		<div class="row">
			<div class="col-12 col-md-4 mt-3 mb-3 mb-md-0">
				<Vehicle v-bind="vehicle"></Vehicle>
			</div>

			<div class="col-12 col-md-4 d-none d-md-block mt-3" v-if="multi">
				<div class="mb-2 pb-1">Modus</div>
				<Mode
					class="btn-group-sm"
					:mode="mode"
					:pvConfigured="pvConfigured"
					v-on:updated="setTargetMode"
				></Mode>
			</div>
			<div class="col-12 col-md-4 d-none d-md-block mt-3" v-if="multi && hasTargetSoC">
				<div class="mb-2 pb-1">Ladeziel</div>
				<Soc
					class="btn-group-sm"
					:soc="targetSoC"
					:levels="socLevels"
					v-on:updated="setTargetSoC"
				></Soc>
			</div>

			<div class="col-md-8 d-none d-md-block" v-if="!multi">
				<LoadpointDetails v-bind="details"></LoadpointDetails>
			</div>

			<div class="col-12 d-md-none">
				<LoadpointDetails v-bind="details"></LoadpointDetails>
			</div>
		</div>
	</div>
</template>

<script>
import axios from "axios";
import Soc from "./Soc";
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
	components: { LoadpointDetails, Soc, Mode, Vehicle },
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
	},
	destroyed: function () {
		window.clearInterval(this.tickerHandle);
	},
};
</script>
