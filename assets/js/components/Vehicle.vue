<template>
	<div>
		<div class="mb-3">
			<div>
				{{ vehicleTitle || $t("main.vehicle.fallbackName") }}
				<span v-if="vehicleProviderLoggedIn">
					<button
						type="button"
						class="btn btn-outline-danger btn-sm"
						@click="providerLogout"
					>
						Logout
					</button>
				</span>
				<span>
					<button
						v-if="!vehicleProviderLoggedIn && vehicleLoginButtonText !== ''"
						type="button"
						class="btn btn-outline-success btn-sm"
						@click="providerLogin"
					>
						{{ vehicleLoginButtonText }}
					</button>
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

import axios from "axios";

import VehicleSoc from "./VehicleSoc";
import VehicleSubline from "./VehicleSubline";

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
		// TODO: Handle language support
		vehicleLoginButtonText: function () {
			if (this.vehicleProviderLoginPath !== "") {
				return "Login";
			}

			return "";
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
			await axios
				.post(this.vehicleProviderLoginPath)
				.then(function (response) {
					window.location.href = response.data.loginUri;
				})
				.catch(function (error) {
					console.log("login failed ", error);
				});
		},
		providerLogout: async function () {
			await axios
				.post(this.vehicleProviderLogoutPath)
				.then(function () {})
				.catch(function (error) {
					console.log("logout failed ", error);
				});
		},
	},
};
</script>
