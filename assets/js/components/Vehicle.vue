<template>
	<div>
		<div class="mb-3">
			<div>
				{{ socTitle || "Fahrzeug" }}
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
		<VehicleSoc v-bind="vehicleSoc" @target-soc-updated="targetSocUpdated" />
		<VehicleSubline v-bind="vehicleSubline" class="my-1" />
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
	props: {
		connected: Boolean,
		hasVehicle: Boolean,
		socCharge: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		socTitle: String,
		timerActive: Boolean,
		timerSet: Boolean,
		targetTime: String,
		targetSoC: Number,
		vehicleProviderLoggedIn: Boolean,
		vehicleProviderLoginPath: String,
		vehicleProviderLogoutPath: String,
	},
	computed: {
		vehicleSoc: function () {
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
	mixins: [collector],
};
</script>
