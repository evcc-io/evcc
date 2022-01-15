<template>
	<div>
		<div class="mb-3">
			<div>
				{{ vehicleTitle || $t("main.vehicle.fallbackName") }}
				<span class="" v-if="showLogin">
					<span v-if="!vehicleProviderLoggedIn">
						<button
							v-if="!vehicleProviderLoggedIn"
							type="button"
							class="btn btn-outline-success btn-sm"
							@click="providerLogin"
						>
							{{ $t("main.provider.login") }}
						</button>
					</span>
					<span v-else>
						<button
							type="button"
							class="btn btn-outline-danger btn-sm"
							@click="providerLogout"
						>
							{{ $t("main.provider.logout") }}
						</button>
					</span>
				</span>
			</div>
		</div>
		<VehicleSoc v-bind="vehicleSocProps" @target-soc-updated="targetSocUpdated" />
		<VehicleSubline
			v-bind="vehicleSubline"
			class="my-1"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
		/>
	</div>
</template>

<script>
import collector from "../mixins/collector";

import auth from "../api";

import VehicleSoc from "./VehicleSoc";
import VehicleSubline from "./VehicleSubline";
import func from 'vue-editor-bridge';

export default {
	name: "Vehicle",
	components: { VehicleSoc, VehicleSubline },
	mixins: [collector],
	props: {
		id: Number,
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		vehicleTitle: String,
		targetTimeActive: Boolean,
		targetTimeHourSuggestion: Number,
		targetTime: String,
		targetSoC: Number,
		vehicleProviderLoggedIn: Boolean,
		vehicleProviderLoginPath: String,
		vehicleProviderLogoutPath: String,
	},
	computed: {
		vehicleSocProps: function () {
			return this.collectProps(VehicleSoc);
		},
		vehicleSubline: function () {
			return this.collectProps(VehicleSubline);
		},
		showLogin: function () {
			return this.vehicleProviderLoginPath && this.vehicleProviderLogoutPath;
		},
	},
	methods: {
		targetSocUpdated: function (targetSoC) {
			this.$emit("target-soc-updated", targetSoC);
		},
		setTargetTime: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
		},
		removeTargetTime: function () {
			this.$emit("target-time-removed");
		},
		providerLogin: async function () {
			auth.post(this.vehicleProviderLoginPath).then(function (response) {
				window.location.href = response.data.loginUri;
			});
		},
		providerLogout: async function () {
			auth.post(this.vehicleProviderLogoutPath)
		},
	},
};
</script>
