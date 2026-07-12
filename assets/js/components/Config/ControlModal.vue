<template>
	<GenericModal
		id="controlModal"
		ref="modal"
		:title="$t('config.control.title')"
		config-modal-name="control"
		data-testid="control-modal"
		@open="open"
	>
		<p>{{ $t("config.control.description") }}</p>
		<p v-if="error" class="text-danger">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
			<FormRow
				id="controlInterval"
				:label="$t('config.control.labelInterval')"
				:help="$t('config.control.descriptionInterval')"
				example="30 s"
				docsLink="/docs/reference/configuration/interval"
			>
				<div class="input-group input-width">
					<input
						id="controlInterval"
						v-model="values.interval"
						type="number"
						step="1"
						min="1"
						required
						aria-describedby="controlIntervalUnit"
						class="form-control text-end"
					/>
					<span id="controlIntervalUnit" class="input-group-text">s</span>
				</div>
			</FormRow>

			<FormRow
				id="controlResidualPower"
				:label="$t('config.control.labelResidualPower')"
				:help="$t('config.control.descriptionResidualPower')"
				example="100 W"
				docsLink="/docs/reference/configuration/site#residualpower"
			>
				<div class="input-group input-width">
					<input
						id="controlResidualPower"
						v-model="values.residualPower"
						type="number"
						step="1"
						required
						aria-describedby="controlResidualPowerUnit"
						class="form-control text-end"
					/>
					<span id="controlResidualPowerUnit" class="input-group-text">W</span>
				</div>
			</FormRow>

			<FormRow
				id="controlPriorityStrategy"
				:label="$t('config.control.labelPriorityStrategy')"
				:help="$t('config.control.descriptionPriorityStrategy')"
			>
				<select
					id="controlPriorityStrategy"
					v-model="values.priorityStrategy"
					class="form-select input-width"
				>
					<option value="none">
						{{ $t("config.control.priorityStrategyNone") }}
					</option>
					<option value="soc">
						{{ $t("config.control.priorityStrategySoc") }}
					</option>
					<option value="deficit">
						{{ $t("config.control.priorityStrategyDeficit") }}
					</option>
				</select>
			</FormRow>

			<FormRow
				v-if="priorityStrategyActive"
				id="controlPriorityBasis"
				:label="$t('config.control.labelPriorityBasis')"
				:help="$t('config.control.descriptionPriorityBasis')"
			>
				<select
					id="controlPriorityBasis"
					v-model="values.priorityBasis"
					class="form-select input-width"
				>
					<option value="percent">
						{{ $t("config.control.priorityBasisPercent") }}
					</option>
					<option value="energy">
						{{ $t("config.control.priorityBasisEnergy") }}
					</option>
				</select>
			</FormRow>

			<FormRow
				v-if="priorityStrategyActive"
				id="controlPriorityHysteresis"
				:label="$t('config.control.labelPriorityHysteresis')"
				:help="$t('config.control.descriptionPriorityHysteresis')"
			>
				<div class="input-group input-width">
					<input
						id="controlPriorityHysteresis"
						v-model="values.priorityHysteresis"
						type="number"
						step="1"
						min="0"
						max="99"
						required
						aria-describedby="controlPriorityHysteresisUnit"
						class="form-control text-end"
					/>
					<span id="controlPriorityHysteresisUnit" class="input-group-text">{{
						priorityHysteresisUnit
					}}</span>
				</div>
			</FormRow>

			<div class="mt-4 d-flex justify-content-between gap-2 flex-column flex-sm-row">
				<button
					type="button"
					class="btn btn-link text-muted btn-cancel"
					data-bs-dismiss="modal"
				>
					{{ $t("config.general.cancel") }}
				</button>

				<button
					type="submit"
					class="btn btn-primary order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
					:disabled="saving || nothingChanged"
				>
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.general.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script>
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "./FormRow.vue";
import store from "@/store";
import api from "@/api";

export default {
	name: "ControlModal",
	components: { FormRow, GenericModal },
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			values: {},
			serverValues: {},
		};
	},
	computed: {
		intervalChanged() {
			return this.values.interval !== this.serverValues.interval;
		},
		residualPowerChanged() {
			return this.values.residualPower !== this.serverValues.residualPower;
		},
		priorityStrategyChanged() {
			return this.values.priorityStrategy !== this.serverValues.priorityStrategy;
		},
		priorityBasisChanged() {
			return this.values.priorityBasis !== this.serverValues.priorityBasis;
		},
		priorityHysteresisChanged() {
			return this.values.priorityHysteresis !== this.serverValues.priorityHysteresis;
		},
		priorityStrategyActive() {
			// basis and hysteresis only affect soc/deficit sub-ordering, not the none strategy
			return this.values.priorityStrategy !== "none";
		},
		priorityHysteresisUnit() {
			return this.values.priorityBasis === "energy" ? "kWh" : "%";
		},
		nothingChanged() {
			return (
				!this.intervalChanged &&
				!this.residualPowerChanged &&
				!this.priorityStrategyChanged &&
				!this.priorityBasisChanged &&
				!this.priorityHysteresisChanged
			);
		},
	},
	methods: {
		reset() {
			const { interval, residualPower, priorityStrategy, priorityBasis, priorityHysteresis } =
				store?.state || {};
			this.saving = false;
			this.error = "";
			this.values = {
				interval,
				residualPower,
				// fall back to the none/percent defaults
				priorityStrategy: priorityStrategy || "none",
				priorityBasis: priorityBasis || "percent",
				priorityHysteresis: priorityHysteresis ?? 0,
			};
			this.serverValues = { ...this.values };
		},
		async open() {
			this.reset();
		},
		async saveValue(name) {
			let url = "";
			if (name === "interval") {
				url = `/config/interval/${encodeURIComponent(this.values.interval)}`;
			} else if (name === "residualPower") {
				url = `/residualpower/${encodeURIComponent(this.values.residualPower)}`;
			} else if (name === "priorityStrategy") {
				url = `/prioritystrategy/${encodeURIComponent(this.values.priorityStrategy)}`;
			} else if (name === "priorityBasis") {
				url = `/prioritybasis/${encodeURIComponent(this.values.priorityBasis)}`;
			} else if (name === "priorityHysteresis") {
				url = `/priorityhysteresis/${encodeURIComponent(this.values.priorityHysteresis)}`;
			}
			await api.post(url);
		},
		async save() {
			this.saving = true;
			this.error = "";
			try {
				if (this.intervalChanged) {
					await this.saveValue("interval");
				}
				if (this.residualPowerChanged) {
					await this.saveValue("residualPower");
				}
				if (this.priorityStrategyChanged) {
					await this.saveValue("priorityStrategy");
				}
				if (this.priorityBasisChanged) {
					await this.saveValue("priorityBasis");
				}
				if (this.priorityHysteresisChanged) {
					await this.saveValue("priorityHysteresis");
				}
				this.$emit("changed");
				this.$refs.modal.close();
			} catch (e) {
				this.error = e.message;
			}
			this.saving = false;
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
.input-width {
	width: 140px;
}
</style>
