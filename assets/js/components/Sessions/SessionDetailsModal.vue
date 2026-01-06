<template>
	<GenericModal
		id="sessionDetailsModal"
		ref="modal"
		:title="$t('session.title')"
		data-testid="session-details"
	>
		<div v-if="session">
			<table class="table align-middle">
				<tbody>
					<tr>
						<th>
							<label for="sessionDetailsLoadpoint">
								{{ $t("sessions.loadpoint") }}
							</label>
						</th>
						<td>
							<CustomSelect
								id="sessionDetailsLoadpoint"
								class="options"
								:options="loadpointOptions"
								:selected="session.loadpoint"
								@change="changeLoadpoint($event.target.value)"
							>
								<span class="flex-grow-1 text-truncate loadpoint-name">
									{{ session.loadpoint || $t("main.loadpoint.fallbackName") }}
								</span>
							</CustomSelect>
						</td>
					</tr>
					<tr>
						<th>
							<label for="sessionDetailsVehicle">
								{{ $t("sessions.vehicle") }}
							</label>
						</th>
						<td>
							<VehicleOptions
								id="sessionDetailsVehicle"
								class="options"
								:vehicleOptions="vehicleOptions"
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
					<tr data-testid="session-details-date">
						<th class="align-baseline">
							{{ $t("session.date") }}
						</th>
						<td>
							<template v-if="session.created">
								{{ fmtFullDateTime(new Date(session.created), false) }}
							</template>
							<br />
							<template v-if="session.finished">
								{{ fmtFullDateTime(new Date(session.finished), false) }}
							</template>
						</td>
					</tr>
					<tr data-testid="session-details-energy">
						<th class="align-baseline">
							{{ $t("sessions.energy") }}
						</th>
						<td>
							{{
								fmtWh(
									chargedEnergy,
									chargedEnergy >= 1e3 ? POWER_UNIT.KW : POWER_UNIT.AUTO
								)
							}}
							<div v-if="session.chargeDuration">
								{{ fmtDurationNs(session.chargeDuration) }}
								(~{{ fmtW(avgPower) }})
							</div>
						</td>
					</tr>
					<tr v-if="session.solarPercentage != null" data-testid="session-details-solar">
						<th class="align-baseline">
							{{ $t("sessions.solar") }}
						</th>
						<td>
							{{ fmtPercentage(session.solarPercentage, 1) }}
							({{ fmtWh(solarEnergy, POWER_UNIT.AUTO) }})
						</td>
					</tr>
					<tr v-if="session.price != null" data-testid="session-details-price">
						<th class="align-baseline">
							{{ $t("session.price") }}
						</th>
						<td>
							{{ fmtMoney(session.price, currency) }}
							{{ fmtCurrencySymbol(currency) }}<br />
							{{ fmtPricePerKWh(session.pricePerKWh || 0, currency) }}
						</td>
					</tr>
					<tr v-if="session.co2PerKWh != null" data-testid="session-details-co2">
						<th>
							{{ $t("session.co2") }}
						</th>
						<td>
							{{ fmtCo2Medium(session.co2PerKWh) }}
						</td>
					</tr>
					<tr v-if="session.odometer" data-testid="session-details-odometer">
						<th>
							{{ $t("session.odometer") }}
						</th>
						<td>
							{{ formatKm(session.odometer) }}
						</td>
					</tr>
					<tr v-if="session.meterStart" data-testid="session-details-meter">
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

			<div class="d-flex justify-content-start">
				<button
					type="button"
					class="btn btn-link text-danger"
					data-testid="session-details-delete"
					@click="openRemoveConfirmationModal"
				>
					{{ $t("session.delete") }}
				</button>
			</div>
		</div>
	</GenericModal>

	<GenericModal
		id="deleteSessionConfirmationModal"
		ref="confirmModal"
		:title="$t('sessions.reallyDelete')"
		data-testid="session-details-confirm"
	>
		<div v-if="session" class="d-flex justify-content-between">
			<button
				type="button"
				class="btn btn-outline-secondary"
				@click="openSessionDetailsModal"
			>
				{{ $t("session.cancel") }}
			</button>
			<button type="button" class="btn btn-danger" @click="removeSession">
				{{ $t("session.delete") }}
			</button>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/checkmark";
import formatter from "@/mixins/formatter";
import Options from "../Vehicles/Options.vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import GenericModal from "../Helper/GenericModal.vue";
import { distanceUnit, distanceValue } from "@/units";
import api from "@/api";
import { defineComponent, type PropType } from "vue";
import type { Session } from "./types";
import type { CURRENCY, SelectOption, Vehicle } from "@/types/evcc";

export default defineComponent({
	name: "SessionDetailsModal",
	components: { VehicleOptions: Options, CustomSelect, GenericModal },
	mixins: [formatter],
	props: {
		session: { type: Object as PropType<Session>, default: () => ({}) },
		currency: { type: String as PropType<CURRENCY> },
		vehicles: { type: Array as PropType<Vehicle[]>, default: () => [] },
		loadpoints: { type: Array as PropType<string[]>, default: () => [] },
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
		vehicleOptions(): SelectOption<string>[] {
			return this.vehicles.map((v) => ({
				name: v.title,
				value: v.title,
			}));
		},
		loadpointOptions(): SelectOption<string>[] {
			return this.loadpoints.map((loadpoint) => ({
				value: loadpoint,
				name: loadpoint,
			}));
		},
	},
	methods: {
		openSessionDetailsModal() {
			(this.$refs["confirmModal"] as any)?.close();
			(this.$refs["modal"] as any)?.open();
		},
		openRemoveConfirmationModal() {
			(this.$refs["modal"] as any)?.close();
			(this.$refs["confirmModal"] as any)?.open();
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
				(this.$refs["confirmModal"] as any)?.close();
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
