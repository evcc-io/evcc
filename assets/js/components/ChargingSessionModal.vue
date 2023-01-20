<template>
	<Teleport to="body">
		<div
			id="sessionDetailsModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
				<div v-if="session" class="modal-content">
					<div class="modal-header">
						<h5>{{ $t("session.title") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<table class="table align-middle">
							<tbody>
								<tr>
									<th>
										{{ $t("sessions.loadpoint") }}
									</th>
									<td>
										{{ session.loadpoint }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("sessions.vehicle") }}
									</th>
									<td>
										<VehicleOptions
											:id="session.vehicle"
											class="options"
											:vehicles="vehicles"
											:is-unknown="false"
											@change-vehicle="changeVehicle"
											@remove-vehicle="removeVehicle"
										>
											<span class="flex-grow-1 text-truncate vehicle-name">
												{{
													session.vehicle
														? session.vehicle
														: $t("main.vehicle.unknown")
												}}
											</span>
										</VehicleOptions>
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.odometer") }}
									</th>
									<td>
										{{ formatKm(session.odometer) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("sessions.energy") }}
									</th>
									<td>
										{{ fmtKWh(session.chargedEnergy * 1e3) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.meterstart") }}
									</th>
									<td>
										{{ fmtKWh(session.meterStart * 1e3) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.meterstop") }}
									</th>
									<td>
										{{ fmtKWh(session.meterStop * 1e3) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.started") }}
									</th>
									<td>
										{{ fmtFullDateTime(new Date(session.created), false) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.finished") }}
									</th>
									<td>
										{{ fmtFullDateTime(new Date(session.finished), false) }}
									</td>
								</tr>
							</tbody>
						</table>
					</div>
					<div class="modal-footer d-flex justify-content-right">
						<button
							type="button"
							class="btn btn-outline-danger"
							data-bs-dismiss="modal"
							@click="openRemoveConfirmationModal"
						>
							{{ $t("session.delete") }}
						</button>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
	<Teleport to="body">
		<div
			id="deleteSessionConfirmationModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
				<div v-if="session" class="modal-content">
					<div class="modal-header">
						<h5>{{ $t("sessions.reallyDelete") }}</h5>
					</div>
					<div class="modal-footer d-flex justify-content-between">
						<button
							type="button"
							class="btn btn-outline-secondary"
							data-bs-dismiss="modal"
							@click="openSessionDetailsModal"
						>
							{{ $t("session.cancel") }}
						</button>
						<button
							type="button"
							class="btn btn-danger"
							data-bs-dismiss="modal"
							@click="removeSession"
						>
							{{ $t("session.delete") }}
						</button>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import "@h2d2/shopicons/es/regular/checkmark";
import { distanceUnit, distanceValue } from "../units";
import formatter from "../mixins/formatter";
import Modal from "bootstrap/js/dist/modal";
import api from "../api";

import VehicleOptions from "./VehicleOptions.vue";

export default {
	name: "ChargingSessionModal",
	components: { VehicleOptions },
	mixins: [formatter],
	props: {
		session: Object,
		vehicles: [Object],
	},
	emits: ["session-changed"],
	methods: {
		openSessionDetailsModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("sessionDetailsModal"));
			modal.show();
		},
		openRemoveConfirmationModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("deleteSessionConfirmationModal")
			);
			modal.show();
		},
		formatKm: function (value) {
			return `${distanceValue(value)} ${distanceUnit()}`;
		},
		async changeVehicle(index) {
			await this.updateSession({
				vehicle: this.vehicles[index - 1].title,
			});
		},
		async removeVehicle() {
			await this.updateSession({
				vehicle: null,
			});
		},
		async updateSession(data) {
			try {
				await api.put("session/" + this.session.id, data);
				this.$emit("session-changed");
			} catch (err) {
				console.error(err);
			}
		},
		async removeSession() {
			try {
				await api.delete("session/" + this.session.id);
				this.$emit("session-changed");
			} catch (err) {
				console.error(err);
			}
		},
	},
};
</script>

<style scoped>
.options .vehicle-name {
	text-decoration: underline;
}
</style>
