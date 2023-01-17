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
										{{ newSession.loadpoint }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("sessions.vehicle") }}
									</th>
									<td>
										<select v-model="newSession.vehicle" class="form-select">
											<option
												v-for="vehicle in vehicles"
												:key="vehicle"
												:value="vehicle"
												:selected="vehicle == newSession.vehicle"
											>
												{{ vehicle }}
											</option>
										</select>
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.odometer") }}
									</th>
									<td>
										{{ formatKm(newSession.odometer) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("sessions.energy") }}
									</th>
									<td>
										{{ fmtKWh(newSession.chargedEnergy * 1e3) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.meterstart") }}
									</th>
									<td>
										{{ fmtKWh(newSession.meterStart * 1e3) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.meterstop") }}
									</th>
									<td>
										{{ fmtKWh(newSession.meterStop * 1e3) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.started") }}
									</th>
									<td>
										{{ fmtFullDateTime(new Date(newSession.created), false) }}
									</td>
								</tr>
								<tr>
									<th>
										{{ $t("session.finished") }}
									</th>
									<td>
										{{ fmtFullDateTime(new Date(newSession.finished), false) }}
									</td>
								</tr>
							</tbody>
						</table>
					</div>
					<div class="modal-footer d-flex justify-content-right">
						<div v-if="sessionUpdated">
							<button
								type="button"
								class="btn btn-outline-warning"
								@click="resetSessionData"
							>
								{{ $t("session.reset") }}
							</button>
							<button
								type="button"
								class="btn btn-outline-success ms-1"
								data-bs-dismiss="modal"
								@click="updateSession"
							>
								{{ $t("session.save") }}
							</button>
						</div>
						<div v-else>
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
import store from "../store";

export default {
	name: "ChargingSessionModal",
	components: {},
	mixins: [formatter],
	props: {
		session: Object,
	},
	emits: ["reload-sessions"],
	data: function () {
		return {
			newSession: undefined,
		};
	},
	computed: {
		sessionUpdated: function () {
			return this.session.vehicle != this.newSession.vehicle;
		},
		vehicles: function () {
			return [...store.state.vehicles, this.$t("main.vehicle.unknown")];
		},
	},
	watch: {
		session: function (session) {
			this.newSession = Object.assign({}, session);
		},
	},
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
		async removeSession() {
			try {
				await api.delete("sessions/" + this.session.id);
				this.$emit("reload-sessions");
			} catch (err) {
				console.error(err);
			}
		},
		async updateSession() {
			try {
				await api.put("sessions", this.newSession);
				this.$emit("reload-sessions");
			} catch (err) {
				console.error(err);
			}
		},
		formatKm: function (value) {
			return `${distanceValue(value)} ${distanceUnit()}`;
		},
		resetSessionData: function () {
			this.newSession = Object.assign({}, this.session);
		},
	},
};
</script>
<style scoped></style>
