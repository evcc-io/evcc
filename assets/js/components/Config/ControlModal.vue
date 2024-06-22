<template>
	<GenericModal
		id="controlModal"
		ref="modal"
		:title="$t('config.control.title')"
		data-testid="control-modal"
		@open="open"
	>
		<p>{{ $t("config.control.description") }}</p>
		<p class="text-danger" v-if="error">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
			<FormRow
				id="controlInterval"
				:label="$t('config.control.labelInterval')"
				:help="$t('config.control.descriptionInterval')"
				example="30 s"
				docsLink="/docs/reference/configuration/interval"
			>
				<div class="input-group w-50 w-sm-25">
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
				<div class="input-group w-50 w-sm-25">
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
				id="controlMaxGridSupply"
				:label="$t('config.control.labelMaxGridSupply')"
				:help="$t('config.control.descriptionMaxGridSupply')"
				docsLink="/docs/reference/configuration/site#maxgridsupplywhilebatterycharging"
			>
				<div class="input-group w-50 w-sm-25">
					<input
						id="controlMaxGridSupply"
						v-model="values.maxGridSupply"
						type="number"
						step="1"
						required
						aria-describedby="controlMaxGridSupplyUnit"
						class="form-control text-end"
					/>
					<span id="controlMaxGridSupplyUnit" class="input-group-text">W</span>
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
import GenericModal from "./../GenericModal.vue";
import FormRow from "./FormRow.vue";
import store from "../../store";
import api from "../../api";

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
		maxGridSupplyChanged() {
			return this.values.maxGridSupply !== this.serverValues.maxGridSupply;
		},
		nothingChanged() {
			return (
				!this.intervalChanged && !this.residualPowerChanged && !this.maxGridSupplyChanged
			);
		},
	},
	methods: {
		reset() {
			const {
				interval,
				residualPower,
				maxGridSupplyWhileBatteryCharging: maxGridSupply,
			} = store?.state || {};
			this.saving = false;
			this.error = "";
			this.values = { interval, residualPower, maxGridSupply };
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
			} else if (name === "maxGridSupply") {
				url = `/maxgridsupply/${encodeURIComponent(this.values.maxGridSupply)}`;
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
				if (this.maxGridSupplyChanged) {
					await this.saveValue("maxGridSupply");
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
.btn-cancel {
	margin-left: -0.75rem;
}
</style>
