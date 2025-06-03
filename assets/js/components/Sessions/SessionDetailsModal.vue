<template>
	<Teleport to="body">
		<div
			id="sessionDetailsModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
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
										<CustomSelect
											id="session.vehicle"
											class="options"
											:options="loadpointOptions"
											:selected="session.loadpoint"
											@change="changeLoadpoint($event.target.value)"
										>
											<span class="flex-grow-1 text-truncate loadpoint-name">
												{{
													session.loadpoint
														? session.loadpoint
														: $t("main.loadpoint.fallbackName")
												}}
											</span>
										</CustomSelect>
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
											:vehicles="vehicleOptions"
											connected
											:selected="session.vehicle"
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
									<th class="align-baseline">
										{{ $t("session.date") }}
									</th>
									<td>
										{{ fmtFullDateTime(session.created, false) }}
										<br />
										{{ fmtFullDateTime(session.finished, false) }}
									</td>
								</tr>
								<tr>
									<th class="align-baseline">
										{{ $t("sessions.energy") }}
									</th>
									<td>
										{{
											fmtWh(
												chargedEnergy,
												chargedEnergy >= 1e3
													? POWER_UNIT.KW
													: POWER_UNIT.AUTO
											)
										}}
										<div v-if="session.chargeDuration">
											{{ fmtDurationNs(session.chargeDuration) }}
											(~{{ fmtW(avgPower) }})
										</div>
									</td>
								</tr>
								<tr v-if="session.solarPercentage != null">
									<th class="align-baseline">
										{{ $t("sessions.solar") }}
									</th>
									<td>
										{{ fmtPercentage(session.solarPercentage, 1) }}
										({{ fmtWh(solarEnergy, POWER_UNIT.AUTO) }})
									</td>
								</tr>
								<tr v-if="session.price != null">
									<th class="align-baseline">
										{{ $t("session.price") }}
									</th>
									<td>
										{{ fmtMoney(session.price, currency) }}
										{{ fmtCurrencySymbol(currency) }}<br />
										{{ fmtPricePerKWh(session.pricePerKWh || 0, currency) }}
									</td>
								</tr>
								<tr v-if="session.co2PerKWh != null">
									<th>
										{{ $t("session.co2") }}
									</th>
									<td>
										{{ fmtCo2Medium(session.co2PerKWh) }}
									</td>
								</tr>
								<tr v-if="session.odometer">
									<th>
										{{ $t("session.odometer") }}
									</th>
									<td>
										{{ formatKm(session.odometer) }}
									</td>
								</tr>
								<tr v-if="session.meterStart">
									<th class="align-baseline">
										{{ $t("session.meter") }}
									</th>
									<td>
										{{ fmtWh(session.meterStart * 1e3) }}<br />
										{{ fmtWh(session.meterStop * 1e3) }}
									</td>
								</tr>
							</tbody>
						</table>
					</div>
					<div class="modal-footer d-flex justify-content-start">
						<button
							type="button"
							class="btn btn-link text-danger"
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
			<div class="modal-dialog modal-dialog-centered" role="document">
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

<script lang="ts">
import "@h2d2/shopicons/es/regular/checkmark";
import Modal from "bootstrap/js/dist/modal";
import formatter from "@/mixins/formatter";
import Options from "../Vehicles/Options.vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import { distanceUnit, distanceValue } from "@/units";
import api from "@/api";
import { defineComponent, type PropType } from "vue";
import type { Session } from "./types";
import type { CURRENCY, LoadpointCompact, SelectOption, Vehicle } from "@/types/evcc";

export default defineComponent({
	name: "SessionDetailsModal",
	components: { VehicleOptions: Options, CustomSelect },
	mixins: [formatter],
	props: {
		session: { type: Object as PropType<Session>, default: () => ({}) },
		currency: { type: String as PropType<CURRENCY> },
		vehicles: { type: Array as PropType<Vehicle[]>, default: () => [] },
		loadpoints: { type: Array as PropType<LoadpointCompact[]>, default: () => [] },
	},
	emits: ["session-changed"],
	computed: {
		chargedEnergy() {
			return this.session.chargedEnergy * 1e3;
		},
		avgPower() {
			const hours = this.session.chargeDuration / 1e9 / 3600;
			return this.chargedEnergy / hours;
		},
		solarEnergy() {
			return this.chargedEnergy * (this.session.solarPercentage / 100);
		},
		vehicleOptions() {
			return this.vehicles.map((v) => ({
				name: v.title,
				title: v.title,
			}));
		},
		loadpointOptions(): SelectOption<string>[] {
			return this.loadpoints.map((loadpoint) => ({
				value: loadpoint.title,
				name: loadpoint.title,
			}));
		},
	},
	methods: {
		openSessionDetailsModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("sessionDetailsModal") as HTMLElement
			);
			modal.show();
		},
		openRemoveConfirmationModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("deleteSessionConfirmationModal") as HTMLElement
			);
			modal.show();
		},
		formatKm(value: number) {
			return `${this.fmtNumber(distanceValue(value), 0)} ${distanceUnit()}`;
		},
		async changeVehicle(title: string) {
			await this.updateSession({ vehicle: title });
		},
		async removeVehicle() {
			await this.updateSession({ vehicle: null });
		},
		async changeLoadpoint(title: string) {
			await this.updateSession({ loadpoint: title });
		},
		async updateSession(data: Partial<Session> | { vehicle: null }) {
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
});
</script>

<style scoped>
.options .vehicle-name {
	text-decoration: underline;
}

.options .loadpoint-name {
	text-decoration: underline;
}
</style>
