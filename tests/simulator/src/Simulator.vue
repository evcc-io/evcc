<template>
	<!-- Mock Login View -->
	<div v-if="mockLoginMode" class="container" style="max-width: 400px; margin-top: 5rem">
		<div class="alert alert-info mb-4">
			<strong>Mock External Login Page</strong>
			<p class="mb-0">This simulates an external OAuth provider login screen.</p>
		</div>

		<h1 class="mb-4">Select Action</h1>

		<button type="button" class="btn btn-success w-100 mb-3" @click="mockLogin('demo-token')">
			Login Successfully
		</button>

		<button type="button" class="btn btn-danger w-100 mb-3" @click="mockDeny()">
			Deny Access
		</button>
	</div>

	<!-- Regular Simulator View -->
	<form
		v-else-if="state"
		class="container"
		style="max-width: 500px; margin-bottom: 8rem"
		@submit.prevent="save"
	>
		<h1 class="my-5 text-center">evcc Simulator</h1>
		<h4 class="my-4">Site</h4>
		<div class="row">
			<label for="gridPower" class="col-sm-6 col-form-label">Grid Power</label>
			<div class="col-sm-6">
				<div class="input-group mb-3">
					<input
						id="gridPower"
						v-model.number="state.site.grid.power"
						type="number"
						class="form-control"
					/>
					<span class="input-group-text">W</span>
				</div>
			</div>
		</div>

		<div class="row">
			<label for="pvPower" class="col-sm-6 col-form-label">PV Power</label>
			<div class="col-sm-6">
				<div class="input-group mb-3">
					<input
						id="pvPower"
						v-model.number="state.site.pv.power"
						type="number"
						class="form-control"
					/>
					<span class="input-group-text">W</span>
				</div>
			</div>
		</div>

		<div class="row">
			<label for="batteryPower" class="col-sm-6 col-form-label">Battery Power</label>
			<div class="col-sm-6">
				<div class="input-group mb-3">
					<input
						id="batteryPower"
						v-model.number="state.site.battery.power"
						type="number"
						class="form-control"
					/>
					<span class="input-group-text">W</span>
				</div>
			</div>
		</div>

		<div class="row">
			<label for="batterySoc" class="col-sm-6 col-form-label">Battery SoC</label>
			<div class="col-sm-6">
				<div class="input-group mb-3">
					<input
						id="batterySoc"
						v-model.number="state.site.battery.soc"
						type="number"
						class="form-control"
					/>
					<span class="input-group-text">%</span>
				</div>
			</div>
		</div>
		<h4 class="my-4">Loadpoints</h4>
		<div
			v-for="(loadpoint, index) in state.loadpoints"
			:key="index"
			class="mb-3"
			:data-testid="`loadpoint${index}`"
		>
			<div class="d-flex justify-content-between my-2">
				<h5>Loadpoint #{{ index }}</h5>
				<a
					v-if="index > 0"
					class="link-danger"
					href="#"
					@click.prevent="removeLoadpoint(index)"
				>
					delete
				</a>
			</div>
			<div class="row">
				<label :for="`loadpointPower${index}`" class="col-sm-6 col-form-label">
					Power
				</label>
				<div class="col-sm-6">
					<div class="input-group mb-3">
						<input
							:id="`loadpointPower${index}`"
							v-model.number="loadpoint.power"
							type="number"
							class="form-control"
						/>
						<span class="input-group-text">W</span>
					</div>
				</div>
			</div>
			<div class="row">
				<label :for="`loadpointEnergy${index}`" class="col-sm-6 col-form-label">
					Energy
				</label>
				<div class="col-sm-6">
					<div class="input-group mb-3">
						<input
							:id="`loadpointEnergy${index}`"
							v-model.number="loadpoint.energy"
							type="number"
							class="form-control"
						/>
						<span class="input-group-text">kWh</span>
					</div>
				</div>
			</div>
			<div class="row">
				<label :for="`loadpointStatus${index}`" class="col-sm-6 col-form-label">
					Status
				</label>
				<div class="col-sm-6 mb-3">
					<div class="form-check">
						<input
							:id="`loadpointStatus${index}1`"
							v-model="loadpoint.status"
							class="form-check-input"
							type="radio"
							value="A"
						/>
						<label :for="`loadpointStatus${index}1`" class="form-check-label">
							A (disconnected)
						</label>
					</div>
					<div class="form-check">
						<input
							:id="`loadpointStatus${index}2`"
							v-model="loadpoint.status"
							class="form-check-input"
							type="radio"
							value="B"
						/>
						<label :for="`loadpointStatus${index}2`" class="form-check-label">
							B (connected)
						</label>
					</div>
					<div class="form-check">
						<input
							:id="`loadpointStatus${index}3`"
							v-model="loadpoint.status"
							class="form-check-input"
							type="radio"
							value="C"
						/>
						<label :for="`loadpointStatus${index}3`" class="form-check-label">
							C (charging)
						</label>
					</div>
				</div>
			</div>
			<div class="row">
				<label :for="`loadpointEnabled${index}`" class="col-sm-6 col-form-label">
					Enabled
				</label>
				<div class="col-sm-6 mb-3">
					<div class="form-check form-switch">
						<input
							:id="`loadpointEnabled${index}`"
							v-model="loadpoint.enabled"
							class="form-check-input"
							type="checkbox"
							role="switch"
						/>
						<label class="form-check-label" :for="`loadpointEnabled${index}`">
							{{ loadpoint.enabled ? "true" : "false" }}
						</label>
					</div>
				</div>
			</div>
		</div>
		<div class="text-end">
			<a class="link-primary" href="#" @click.prevent="addLoadpoint"> add loadpoint </a>
		</div>

		<h4 class="my-4">Vehicles</h4>
		<div
			v-for="(vehicle, index) in state.vehicles"
			:key="index"
			class="mb-3"
			:data-testid="`vehicle${index}`"
		>
			<div class="d-flex justify-content-between my-2">
				<h5>Vehicle #{{ index }}</h5>
				<a
					v-if="index > 0"
					class="link-danger"
					href="#"
					@click.prevent="removeVehicle(index)"
				>
					delete
				</a>
			</div>
			<div class="row">
				<label :for="`vehicleSoc${index}`" class="col-sm-6 col-form-label"> SoC </label>
				<div class="col-sm-6">
					<div class="input-group mb-3">
						<input
							:id="`vehicleSoc${index}`"
							v-model.number="vehicle.soc"
							type="number"
							class="form-control"
						/>
						<span class="input-group-text">%</span>
					</div>
				</div>
			</div>
			<div class="row">
				<label :for="`vehicleRange${index}`" class="col-sm-6 col-form-label"> Range </label>
				<div class="col-sm-6">
					<div class="input-group mb-3">
						<input
							:id="`vehicleRange${index}`"
							v-model.number="vehicle.range"
							type="number"
							class="form-control"
						/>
						<span class="input-group-text">km</span>
					</div>
				</div>
			</div>
		</div>
		<div class="text-end">
			<a class="link-primary" href="#" @click.prevent="addVehicle"> add vehicle </a>
		</div>

		<h4 class="my-4">HEMS</h4>
		<div class="row">
			<label for="hemsRelay" class="col-sm-6 col-form-label">Relay Limit</label>
			<div class="col-sm-6 mb-3">
				<div class="form-check form-switch">
					<input
						id="hemsRelay"
						v-model="state.hems.relay"
						class="form-check-input"
						type="checkbox"
						role="switch"
					/>
					<label class="form-check-label" for="hemsRelay"> active </label>
				</div>
			</div>
		</div>

		<h4 class="my-4">OCPP <small>work in progress, connect only</small></h4>

		<!-- Connected Clients -->
		<div
			v-for="client in state.ocpp.clients"
			:key="client.stationId"
			class="card mb-3"
			:data-testid="`ocpp-client-${client.stationId}`"
		>
			<div class="card-body">
				<div class="d-flex justify-content-between align-items-center mb-2">
					<h5 class="card-title mb-0">{{ client.stationId }}</h5>
					<span class="badge" :class="client.connected ? 'bg-success' : 'bg-secondary'">
						{{ client.connected ? "Connected" : "Disconnected" }}
					</span>
				</div>
				<p class="card-text text-muted small mb-2">{{ client.serverUrl }}</p>
				<button
					type="button"
					class="btn btn-sm btn-danger"
					@click="disconnectOcpp(client.stationId)"
				>
					Disconnect
				</button>
			</div>
		</div>

		<!-- Add OCPP Client Card -->
		<div class="card mb-3" data-testid="ocpp-add-client">
			<div class="card-body">
				<h5 class="card-title">OCPP Client</h5>
				<div class="row">
					<label for="ocppServerUrl" class="col-sm-6 col-form-label">Server URL</label>
					<div class="col-sm-6">
						<div class="input-group mb-3">
							<input
								id="ocppServerUrl"
								v-model="ocppServerUrl"
								type="text"
								class="form-control"
							/>
						</div>
					</div>
				</div>
				<div class="row">
					<label for="ocppStationId" class="col-sm-6 col-form-label">Station ID</label>
					<div class="col-sm-6">
						<div class="input-group mb-3">
							<input
								id="ocppStationId"
								v-model="ocppStationId"
								type="text"
								class="form-control"
							/>
						</div>
					</div>
				</div>
				<div class="row">
					<div class="col-sm-6"></div>
					<div class="col-sm-6">
						<button
							type="button"
							class="btn btn-primary w-100"
							:disabled="!ocppServerUrl || !ocppStationId || connecting"
							@click="connectOcpp"
						>
							{{ connecting ? "Connecting..." : "Connect" }}
						</button>
					</div>
				</div>
			</div>
		</div>

		<div class="p-4 text-center fixed-bottom bg-light text-dark bg-opacity-75">
			<button type="submit" class="btn btn-primary">Apply changes</button>
		</div>
	</form>
</template>

<script lang="ts">
import axios from "axios";
import { defineComponent } from "vue";

export default defineComponent({
	name: "Simulator",
	data() {
		return {
			mockLoginMode: false,
			mockLoginState: "",
			mockLoginRedirectUri: "",
			state: null as {
				site: {
					grid: { power: number };
					pv: { power: number; energy: number };
					battery: { power: number; soc: number };
				};
				loadpoints: {
					power: number;
					energy: number;
					enabled: boolean;
					status: string;
				}[];
				vehicles: { soc: number; range: number }[];
				hems: { relay: boolean };
				ocpp: {
					clients: { stationId: string; serverUrl: string; connected: boolean }[];
				};
			} | null,
			ocppServerUrl: "ws://127.0.0.1:8887/",
			ocppStationId: "",
			connecting: false,
		};
	},
	mounted() {
		this.checkMockLoginMode();
		if (!this.mockLoginMode) {
			this.load();
		}
	},
	methods: {
		checkMockLoginMode() {
			const urlParams = new URLSearchParams(window.location.search);
			const state = urlParams.get("state");
			const redirectUri = urlParams.get("redirectUri");

			if (state && redirectUri) {
				this.mockLoginMode = true;
				this.mockLoginState = state;
				this.mockLoginRedirectUri = redirectUri;
			}
		},
		mockLogin(code: string) {
			const callbackUrl = `${this.mockLoginRedirectUri}?state=${this.mockLoginState}&code=${code}`;
			console.log("Redirecting to:", callbackUrl);
			window.location.href = callbackUrl;
		},
		mockDeny() {
			const callbackUrl = `${this.mockLoginRedirectUri}?state=${this.mockLoginState}&error=access_denied&error_description=User denied authorization`;
			console.log("Redirecting to:", callbackUrl);
			window.location.href = callbackUrl;
		},
		async save() {
			await axios.post("/api/state", this.state);
		},
		async load() {
			const response = await axios.get("/api/state");
			this.state = response.data;
		},
		async connectOcpp() {
			this.connecting = true;
			try {
				await axios.post("/api/ocpp/connect", {
					stationId: this.ocppStationId,
					serverUrl: this.ocppServerUrl,
				});
				// Reload state to get updated ocpp status
				await this.load();
				this.ocppStationId = "";
			} catch (error) {
				console.error("Failed to connect OCPP client:", error);
				alert("Failed to connect OCPP client");
			} finally {
				this.connecting = false;
			}
		},
		async disconnectOcpp(stationId: string) {
			try {
				await axios.post("/api/ocpp/disconnect", { stationId });
				// Reload state to get updated ocpp status
				await this.load();
			} catch (error) {
				console.error("Failed to disconnect OCPP client:", error);
				alert("Failed to disconnect OCPP client");
			}
		},
		addVehicle() {
			// push a duplacate of the last entry
			const vehicles = this.state?.vehicles;
			if (!vehicles) return;
			const lastVehicle = vehicles[vehicles.length - 1];
			if (lastVehicle) {
				vehicles.push({ ...lastVehicle });
			}
		},
		removeVehicle(index: number) {
			this.state?.vehicles.splice(index, 1);
		},
		addLoadpoint() {
			// push a duplacate of the last entry
			const loadpoints = this.state?.loadpoints;
			if (!loadpoints) return;
			const lastLoadpoint = loadpoints[loadpoints.length - 1];
			if (lastLoadpoint) {
				loadpoints.push({ ...lastLoadpoint });
			}
		},
		removeLoadpoint(index: number) {
			this.state?.loadpoints.splice(index, 1);
		},
	},
});
</script>

<style>
input[type="number"]::-webkit-outer-spin-button,
input[type="number"]::-webkit-inner-spin-button {
	-webkit-appearance: none;
	margin: 0;
}
input[type="number"] {
	appearance: textfield;
	-moz-appearance: textfield;
}
input[type="number"] {
	text-align: right;
}
</style>
