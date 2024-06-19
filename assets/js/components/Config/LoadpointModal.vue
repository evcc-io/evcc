<template>
	<GenericModal
		id="loadpointModal"
		:title="modalTitle"
		data-testid="loadpoint-modal"
		@open="open"
		@closed="closed"
	>
		<form ref="form" class="container mx-0 px-0" @submit.prevent="isNew ? create() : update()">
			<FormRow id="loadpointParamTitle" label="Title" example="Garage, Carport, etc.">
				<PropertyField
					id="loadpointParamTitle"
					v-model="values.title"
					type="String"
					class="me-2"
					required
				/>
			</FormRow>
			<FormRow id="loadpointParamCharger" label="Charger">
				<div class="d-flex">
					<PropertyField
						id="loadpointParamCharger"
						v-model="values.charger"
						type="String"
						class="me-2 flex-grow-1"
						disabled
						required
					/>
					<button
						class="btn btn-link btn-sm evcc-default-text"
						type="button"
						@click.prevent="editCharger"
					>
						<shopicon-regular-adjust></shopicon-regular-adjust>
					</button>
				</div>
			</FormRow>
			<div v-if="values.charger">
				<FormRow
					v-if="values.meter"
					id="loadpointParamMeter"
					label="Energy meter"
					help="Additional meter if the charger doesn't have an integrated one."
				>
					<div class="d-flex">
						<PropertyField
							id="loadpointParamMeter"
							v-model="values.meter"
							type="String"
							class="me-2 flex-grow-1"
							disabled
							required
						/>
						<button
							class="btn btn-link btn-sm evcc-default-text"
							type="button"
							@click.prevent="editMeter"
						>
							<shopicon-regular-adjust></shopicon-regular-adjust>
						</button>
					</div>
				</FormRow>
				<p v-else>
					<button
						class="btn btn-link btn-sm text-primary px-0"
						type="button"
						@click="editMeter"
					>
						Add dedicated charger meter
					</button>
				</p>
			</div>

			<h6>Basics</h6>

			<div class="row">
				<FormRow
					id="loadpointMode"
					label="Default mode"
					help="Mode when connecting the vehicle."
					class="col-md-6"
				>
					<PropertyField
						id="loadpointMode"
						v-model="values.mode"
						type="Number"
						size="w-75 w-min-200"
						class="me-2"
						:valid-values="[
							{ key: '', name: 'use last' },
							{ key: null, name: null },
							{ key: 'off', name: $t('main.mode.off') },
							{ key: 'pv', name: $t('main.mode.pv') },
							{ key: 'minpv', name: $t('main.mode.minpv') },
							{ key: 'now', name: $t('main.mode.off') },
						]"
						required
					/>
				</FormRow>
				<FormRow
					id="loadpointParamPhases"
					label="Phases"
					help="Electrical connection of the charger."
					class="col-md-6"
				>
					<PropertyField
						id="loadpointParamPhases"
						v-model="values.phases"
						type="Number"
						size="w-75 w-min-200"
						class="me-2"
						:valid-values="[
							{ key: 0, name: 'automatic switching' },
							{ key: null, name: null },
							{ key: 1, name: '1-phase' },
							{ key: 3, name: '3-phase' },
						]"
						required
					/>
				</FormRow>
			</div>

			<FormRow
				id="loadpointParamMaxCurrent"
				label="Current limits"
				:help="
					values.minCurrent < 6
						? 'Currents below 6 A are not supported by electric vehicles. Only use lower values if you know what you are doing.'
						: null
				"
			>
				<CurrentRange v-model:min="values.minCurrent" v-model:max="values.maxCurrent" />
			</FormRow>

			<h6>Solar behaviour</h6>

			<div class="row">
				<FormRow
					id="loadpointParamPriority"
					label="Priority"
					help="Relevant for multiple charge points. Higher priority charge points get preferred access to solar surplus."
				>
					<PropertyField
						id="loadpointParamPriority"
						v-model="values.priority"
						type="Number"
						size="w-100"
						class="me-2"
						:valid-values="priorityOptions"
						required
					/>
				</FormRow>
			</div>

			<div class="row">
				<FormRow id="loadpointEnableDelay" label="Enable Delay" class="col-sm-6">
					<PropertyField
						id="loadpointEnableDelay"
						v-model="values.thresholds.enable.delay"
						type="Duration"
						unit="min"
						size="w-25 w-min-200"
						class="me-2"
						required
					/>
				</FormRow>
				<FormRow id="loadpointDisableDelay" label="Disable Delay" class="col-sm-6">
					<PropertyField
						id="loadpointDisableDelay"
						v-model="values.thresholds.disable.delay"
						type="Duration"
						unit="min"
						size="w-25 w-min-200"
						class="me-2"
						required
					/>
				</FormRow>
			</div>

			<div class="row">
				<FormRow
					id="loadpointEnableThreshold"
					label="Enable Threshold"
					class="col-sm-6 mb-sm-0"
				>
					<PropertyField
						id="loadpointEnableThreshold"
						v-model="values.thresholds.enable.threshold"
						type="Float"
						unit="kW"
						size="w-25 w-min-200"
						class="me-2"
						required
					/>
				</FormRow>

				<FormRow
					id="loadpointDisableThreshold"
					label="Disable Threshold"
					class="col-sm-6 mb-sm-0"
				>
					<PropertyField
						id="loadpointDisableThreshold"
						v-model="values.thresholds.disable.threshold"
						type="Float"
						unit="kW"
						size="w-25 w-min-200"
						class="me-2"
						required
					/>
				</FormRow>
			</div>

			<h6>Vehicle</h6>

			<FormRow
				id="loadpointParamVehicle"
				label="Default vehicle"
				:help="
					values.defaultVehicle
						? 'Always assume this vehicle is charging here. Auto-detection disabled. Manual override is possible.'
						: 'Automatically selects the most plausible vehicle. Manual override is possible.'
				"
			>
				<PropertyField
					id="loadpointParamVehicle"
					v-model="values.defaultVehicle"
					type="String"
					class="me-2"
					:valid-values="allVehicleOptions"
					required
				/>
			</FormRow>

			<div class="row">
				<FormRow
					id="loadpointPollMode"
					label="Poll Mode"
					help="When to update vehicle status information."
					class="col-md-6"
				>
					<PropertyField
						id="loadpointPollMode"
						v-model="values.soc.poll.mode"
						type="Number"
						size="w-75 w-min-200"
						class="me-2"
						:valid-values="[
							{ key: 'pollcharging', name: 'when charging' },
							{ key: 'pollconnected', name: 'when connected' },
							{ key: 'pollalways', name: 'always' },
						]"
						required
					/>
				</FormRow>
				<FormRow
					v-if="values.soc.poll.mode !== 'pollcharging'"
					id="loadpointPollInterval"
					label="Poll Interval"
					help="Time between status updates. Short intervals may drain the vehicle battery."
					class="col-md-6"
				>
					<PropertyField
						id="loadpointPollInterval"
						v-model="values.soc.poll.interval"
						type="Duration"
						unit="min"
						size="w-25 w-min-200"
						class="me-2"
						required
					/>
				</FormRow>
			</div>

			<FormRow id="loadpointEstimate" label="Estimate charge level">
				<div class="d-flex">
					<input
						class="form-check-input"
						id="loadpointEstimate"
						type="checkbox"
						v-model="values.soc.estimate"
					/>
					<label class="form-check-label ms-2" for="loadpointEstimate">
						Interpolate between API updates
					</label>
				</div>
			</FormRow>

			<div class="mt-5 mb-4 d-flex justify-content-between">
				<button
					v-if="isDeletable"
					type="button"
					class="btn btn-link text-danger"
					@click.prevent="remove"
				>
					{{ $t("config.meter.delete") }}
				</button>
				<button
					v-else
					type="button"
					class="btn btn-link text-muted btn-cancel"
					data-bs-dismiss="modal"
				>
					{{ $t("config.loadpoint.cancel") }}
				</button>
				<button type="submit" class="btn btn-primary" :disabled="saving">
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.loadpoint.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script>
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import CurrentRange from "./CurrentRange.vue";
import api from "../../api";
import GenericModal from "../GenericModal.vue";
import deepClone from "../../utils/deepClone";

const nsPerMin = 60 * 1e9;
const wPerKw = 1e3;

const defaultValues = {
	title: "",
	phases: 0,
	minCurrent: 6,
	maxCurrent: 16,
	priority: 0,
	mode: "",
	thresholds: {
		enable: { delay: 1, threshold: 0 },
		disable: { delay: 3, threshold: 0 },
	},
	soc: {
		poll: { mode: "pollcharging", interval: 60 },
		estimate: false,
	},
	defaultVehicle: "",
	charger: "",
	meter: "",
};

export default {
	name: "LoadpointModal",
	components: { FormRow, PropertyField, CurrentRange, GenericModal },
	props: {
		id: Number,
		name: String,
		vehicleOptions: { type: Array, default: () => [] },
	},
	emits: ["updated", "openMeterModal", "openChargerModal"],
	data() {
		return {
			isModalVisible: false,
			saving: false,
			selectedType: null,
			values: deepClone(defaultValues),
		};
	},
	computed: {
		modalTitle() {
			if (this.isNew) {
				return this.$t(`config.loadpoint.titleAdd`);
			}
			return this.$t(`config.loadpoint.titleEdit`);
		},
		isNew() {
			return this.id === undefined;
		},
		isDeletable() {
			return !this.isNew;
		},
		priorityOptions() {
			const result = Array.from({ length: 11 }, (_, i) => ({ key: i, name: `${i}` }));
			result[0].name = "0 (default)";
			result[10].name = "10 (highest)";
			return result;
		},
		allVehicleOptions() {
			return [
				{ key: "", name: "auto detection" },
				{ key: null, name: null },
				...this.vehicleOptions,
			];
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				if (this.values?.id !== this.id) {
					// loadpoint changed
					this.reset();
					if (this.id) {
						this.loadConfiguration();
					}
				}
			}
		},
	},
	methods: {
		reset() {
			this.values = deepClone(defaultValues);
		},
		async loadConfiguration() {
			try {
				const res = await api.get(`config/loadpoints/${this.id}`);
				this.values = this.transformAfterLoad(res.data.result);
			} catch (e) {
				console.error(e);
			}
		},
		async update() {
			this.saving = true;
			try {
				const values = this.transformBeforeSave(this.values);
				await api.put(`config/loadpoints/${this.id}`, values);
				this.$emit("updated");
				this.closed();
			} catch (e) {
				console.error(e);
				alert("update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await api.delete(`config/loadpoints/${this.id}`);
				this.$emit("updated");
				this.closed();
			} catch (e) {
				console.error(e);
				alert("delete failed");
			}
		},
		async create() {
			this.saving = true;
			try {
				await api.post("config/loadpoints", this.values);
				this.$emit("updated");
				this.closed();
			} catch (e) {
				console.error(e);
				alert("create failed");
			}
			this.saving = false;
		},
		open() {
			this.isModalVisible = true;
		},
		closed() {
			this.isModalVisible = false;
		},
		editCharger() {
			this.$emit("openChargerModal", this.values.charger);
		},
		editMeter() {
			this.$emit("openMeterModal", this.values.meter);
		},
		// called externally
		setMeter(meter) {
			this.values.meter = meter;
		},
		// called externally
		setCharger(charger) {
			this.values.charger = charger;
		},
		scaleValueToInt(obj, key, scale) {
			obj[key] = Math.round(obj[key] * scale);
		},
		transformBeforeSave(values) {
			function scale(obj, key, scale) {
				obj[key] = Math.round(obj[key] * scale);
			}

			const result = deepClone(values);
			scale(result.thresholds.enable, "delay", nsPerMin);
			scale(result.thresholds.disable, "delay", nsPerMin);
			scale(result.thresholds.enable, "threshold", wPerKw);
			scale(result.thresholds.disable, "threshold", wPerKw);
			scale(result.soc.poll, "interval", nsPerMin);

			return result;
		},
		transformAfterLoad(values) {
			const result = deepClone(values);
			result.thresholds.enable.delay /= nsPerMin;
			result.thresholds.enable.threshold /= wPerKw;
			result.thresholds.disable.delay /= nsPerMin;
			result.thresholds.disable.threshold /= wPerKw;
			result.soc.poll.interval /= nsPerMin;
			return result;
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
.addButton {
	min-height: auto;
}
.btn-cancel {
	margin-left: -0.75rem;
}
h6 {
	margin-top: 4rem;
}
</style>
