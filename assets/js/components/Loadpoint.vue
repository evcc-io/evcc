<template>
	<div>
		<div class="row" v-if="multi">
			<div class="col-12 col-md-4 d-md-flex mt-3 mt-md-5 align-items-end">
				<span class="h1 align-bottom">{{ state.title || "Ladepunkt" }}</span>
			</div>

			<div class="col-12 col-md-8 d-none d-md-block mt-3 mt-md-5">
				<LoadpointDetails v-bind:state="state"></LoadpointDetails>
			</div>

			<div class="col-12 d-md-none">
				<div class="row mt-3 pb-3 bg-light">
					<div class="col-12 mt-3">
						<Mode
							class="w-100"
							v-bind:mode="state.mode"
							:pv="pv"
							v-on:updated="targetMode"
						></Mode>
					</div>
					<div class="col-12 mt-3" v-if="hasTargetSoC">
						<Soc
							class="w-100"
							v-bind:soc="state.targetSoC"
							:levels="state.socLevels"
							v-on:updated="targetSoC"
						></Soc>
					</div>
				</div>
			</div>
		</div>

		<div class="row d-none d-md-flex mt-5 py-3 pb-4 text-center bg-light" v-if="!multi">
			<div
				class="mt-3"
				v-bind:class="{ 'col-md-6': hasTargetSoC, 'col-md-12': !hasTargetSoC }"
			>
				<Mode
					v-bind:mode="state.mode"
					:pv="pv"
					:caption="true"
					v-on:updated="targetMode"
				></Mode>
			</div>
			<div class="col-md-6 mt-3" v-if="hasTargetSoC">
				<Soc
					v-bind:soc="state.targetSoC"
					:levels="state.socLevels"
					:caption="true"
					v-on:updated="targetSoC"
				></Soc>
			</div>
		</div>

		<div class="row d-md-none mt-2 pb-3 bg-light" v-if="!multi">
			<div class="col-12 mt-3">
				<Mode
					class="w-100"
					v-bind:mode="state.mode"
					:pv="pv"
					v-on:updated="targetMode"
				></Mode>
			</div>
			<div class="col-12 mt-3" v-if="hasTargetSoC">
				<Soc
					class="w-100"
					v-bind:soc="state.targetSoC"
					:levels="state.socLevels"
					v-on:updated="targetSoC"
				></Soc>
			</div>
		</div>

		<div class="row" v-if="!multi">
			<div class="col-12 col-md-4 d-none d-md-flex mt-3 mt-md-5">
				<span class="h1">{{ state.title || "Ladepunkt" }}</span>
			</div>
			<div
				class="col-12 col-md-8 d-flex d-md-flex mt-3 mt-md-5 pt-3"
				v-if="state.remoteDisabled"
			>
				<h5 class="w-100">
					<span class="badge badge-warning w-100" v-if="state.remoteDisabled == 'soft'">
						{{ state.remoteDisabledSource }}: Adaptives PV-Laden deaktiviert
					</span>
					<span class="badge badge-danger w-100" v-if="state.remoteDisabled == 'hard'">
						{{ state.remoteDisabledSource }}: Deaktiviert
					</span>
				</h5>
			</div>
		</div>

		<div class="row border-bottom d-none d-md-block"></div>

		<div class="row">
			<div class="col-12 col-md-4 mt-3 mb-3 mb-md-0">
				<Vehicle v-bind:state="state"></Vehicle>
			</div>

			<div class="col-12 col-md-4 d-none d-md-block mt-3" v-if="multi">
				<div class="mb-2">Modus</div>
				<Mode
					class="btn-group-sm"
					v-bind:mode="state.mode"
					:pv="pv"
					v-on:updated="targetMode"
				></Mode>
			</div>
			<div class="col-12 col-md-4 d-none d-md-block mt-3" v-if="multi && hasTargetSoC">
				<div class="mb-2">Ladeziel</div>
				<Soc
					class="btn-group-sm"
					v-bind:soc="state.targetSoC"
					:levels="state.socLevels"
					v-on:updated="targetSoC"
				></Soc>
			</div>

			<div class="col-md-8 d-none d-md-block" v-if="!multi">
				<LoadpointDetails v-bind:state="state"></LoadpointDetails>
			</div>

			<div class="col-12 d-md-none">
				<LoadpointDetails v-bind:state="state"></LoadpointDetails>
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

export default {
	name: "Loadpoint",
	props: ["state", "id", "pv", "multi"],
	components: { LoadpointDetails, Soc, Mode, Vehicle },
	mixins: [formatter],
	data: function () {
		return {
			tickerHandle: null,
		};
	},
	computed: {
		hasTargetSoC: function () {
			return this.state.socLevels != null && this.state.socLevels.length > 0;
		},
	},
	watch: {
		"state.chargeDuration": function () {
			window.clearInterval(this.tickerHandle);
			// only ticker if actually charging
			if (this.state.charging && this.state.chargeDuration >= 0) {
				this.tickerHandle = window.setInterval(
					function () {
						// eslint-disable-next-line vue/no-mutating-props
						this.state.chargeDuration += 1;
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
		targetMode: function (mode) {
			axios
				.post(this.api("mode") + "/" + mode)
				.then(
					function (response) {
						// eslint-disable-next-line vue/no-mutating-props
						this.state.mode = response.data.mode;
					}.bind(this)
				)
				.catch(window.toasts.error);
		},
		targetSoC: function (soc) {
			axios
				.post(this.api("targetsoc") + "/" + soc)
				.then(
					function (response) {
						// eslint-disable-next-line vue/no-mutating-props
						this.state.targetSoC = response.data.targetSoC;
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
