<template>
	<GenericModal
		id="loadpointModal"
		:title="modalTitle"
		data-testid="loadpoint-modal"
		:fade="fade"
		@open="open"
		@opened="opened"
		@close="close"
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
			<FormRow v-if="values.charger" id="loadpointParamCharger" label="Charger">
				<div class="d-flex">
					<PropertyField
						id="loadpointParamCharger"
						:modelValue="chargerTitle"
						type="String"
						class="me-2 flex-grow-1"
						readonly
						required
						@click.prevent="editCharger"
					/>
					<button
						class="btn btn-link btn-sm evcc-default-text"
						type="button"
						@click.prevent="editCharger"
					>
						<EditIcon />
					</button>
				</div>
			</FormRow>
			<div v-else class="d-flex justify-content-end">
				<button
					class="btn btn-outline-primary"
					type="submit"
					:disabled="values.title?.length === 0"
					@click.prevent="editCharger"
				>
					Add charger
				</button>
			</div>
			<div v-if="values.charger || !isNew">
				<FormRow
					v-if="values.meter"
					id="loadpointParamMeter"
					class="mb-6"
					label="Energy meter"
					help="Additional meter if the charger doesn't have an integrated one."
				>
					<div class="d-flex">
						<PropertyField
							id="loadpointParamMeter"
							:modelValue="meterTitle"
							type="String"
							class="me-2 flex-grow-1"
							required
							readonly
							@click="editMeter"
						/>
						<button
							class="btn btn-link btn-sm evcc-default-text"
							type="button"
							@click.prevent="editMeter"
						>
							<EditIcon />
						</button>
					</div>
				</FormRow>
				<p v-else>
					<button
						class="btn btn-link btn-sm text-gray px-0"
						style="margin-top: -1rem"
						type="button"
						@click="editMeter"
					>
						Add dedicated charger meter
					</button>
				</p>
			</div>

			<div v-if="values.charger">
				<h6>Charging</h6>

				<FormRow
					id="loadpointMode"
					label="Mode"
					help="Charging mode when connecting the vehicle."
				>
					<PropertyField
						id="loadpointMode"
						v-model="values.mode"
						type="String"
						class="w-100"
						required
						:valid-values="[
							{ value: '', name: 'Keep last mode' },
							{ value: 'off', name: $t('main.mode.off') },
							{ value: 'pv', name: $t('main.mode.pv') },
							{ value: 'minpv', name: $t('main.mode.minpv') },
							{ value: 'now', name: $t('main.mode.now') },
						]"
					/>
				</FormRow>

				<FormRow
					id="loadpointSolarMode"
					label="Solar behaviour"
					:help="
						solarMode === 'default'
							? `Only charge with solar surplus. Fast start (${fmtDurationNs(values.thresholds.enable.delay, true, 'm')}) and stop (${fmtDurationNs(values.thresholds.disable.delay, true, 'm')}).`
							: 'Define your own enable and disable thresholds and delays.'
					"
				>
					<SelectGroup
						id="loadpointSolarMode"
						v-model="solarMode"
						class="w-100"
						:options="[
							{ name: 'default', value: 'default' },
							{ name: 'custom', value: 'custom' },
						]"
						transparent
					/>
				</FormRow>

				<div v-show="solarMode === 'custom'" class="ms-3 mb-5">
					<div class="mb-4">
						<div class="d-flex flex-wrap flex-sm-nowrap gap-4">
							<FormRow
								id="loadpointEnableThreshold"
								label="Enable grid power"
								style="margin-bottom: 0 !important"
							>
								<PropertyField
									id="loadpointEnableThreshold"
									v-model="values.thresholds.enable.threshold"
									type="Float"
									unit="W"
									size="w-25 w-min-200"
									required
								/>
							</FormRow>
							<FormRow
								id="loadpointEnableDelay"
								label="Enable delay"
								style="margin-bottom: 0 !important"
							>
								<PropertyField
									id="loadpointEnableDelay"
									v-model="values.thresholds.enable.delay"
									type="Duration"
									unit="minute"
									size="w-25 w-min-200"
									required
								/>
							</FormRow>
						</div>
						<div class="form-text evcc-gray">
							{{
								values.thresholds.enable.threshold === 0
									? `Start when minimum charge power surplus is available for ${fmtDurationNs(values.thresholds.enable.delay, true, "m")}.`
									: values.thresholds.enable.threshold < 0
										? `Start charging, when ${fmtW(-1 * values.thresholds.enable.threshold, powerUnit.AUTO)} surplus is available for ${fmtDurationNs(values.thresholds.enable.delay, true, "m")}.`
										: `Please use a negative value.`
							}}
						</div>
					</div>

					<div>
						<div class="d-flex flex-wrap flex-sm-nowrap gap-4">
							<FormRow
								id="loadpointDisableThreshold"
								label="Disable grid power"
								style="margin-bottom: 0 !important"
							>
								<PropertyField
									id="loadpointDisableThreshold"
									v-model="values.thresholds.disable.threshold"
									type="Float"
									unit="W"
									size="w-25 w-min-200"
									required
								/>
							</FormRow>
							<FormRow
								id="loadpointDisableDelay"
								label="Disable delay"
								style="margin-bottom: 0 !important"
							>
								<PropertyField
									id="loadpointDisableDelay"
									v-model="values.thresholds.disable.delay"
									type="Duration"
									unit="minute"
									size="w-25 w-min-200"
									required
								/>
							</FormRow>
						</div>
						<div class="form-text evcc-gray">
							{{
								values.thresholds.disable.threshold === 0
									? `Stop when minimum charge power can't be satisfied for ${fmtDurationNs(values.thresholds.disable.delay, true, "m")}.`
									: values.thresholds.disable.threshold > 0
										? `Stop charging, when more than ${fmtW(values.thresholds.disable.threshold, powerUnit.AUTO)} is used from the grid for ${fmtDurationNs(values.thresholds.disable.delay, true, "m")}.`
										: `Please use a positive value.`
							}}
						</div>
					</div>
				</div>

				<FormRow
					v-if="showPriority"
					id="loadpointParamPriority"
					label="Priority"
					help="Higher priority charge points get preferred access to solar surplus."
				>
					<SelectGroup
						id="loadpointParamPriority"
						v-model="values.priority"
						class="w-100"
						:options="priorityOptions"
						transparent
					/>
				</FormRow>

				<h6>
					Electrical
					<small class="text-muted">When in doubt, ask your electrician.</small>
				</h6>

				<FormRow
					id="chargerPower"
					label="Charger type"
					:help="
						chargerPower === '11kw'
							? 'Will use a current range of 6 to 16 A.'
							: chargerPower === '22kw'
								? 'Will use a current range of 6 to 32 A.'
								: 'Define a custom current range.'
					"
				>
					<SelectGroup
						id="chargerPower"
						v-model="chargerPower"
						class="w-100"
						:options="[
							{ name: '11 kW', value: '11kw' },
							{ name: '22 kW', value: '22kw' },
							{ name: 'other', value: 'other' },
						]"
						transparent
					/>
				</FormRow>

				<FormRow
					v-if="!is1p3pSupported"
					id="loadpointParamPhases"
					label="Phases"
					help="Number of phases connected to the charger."
				>
					<SelectGroup
						id="loadpointParamPhases"
						v-model="values.phases"
						class="w-100"
						:options="[
							{ value: 1, name: '1-phase' },
							{ value: 3, name: '3-phase' },
						]"
						transparent
					/>
				</FormRow>

				<div v-if="chargerPower === 'other'" class="row ms-3 mb-5">
					<FormRow
						id="loadpointMinCurrent"
						label="Minimum current"
						class="col-sm-6 mb-sm-0"
						:help="
							values.minCurrent < 6
								? 'Only go below 6 A if you know what you\'re doing.'
								: null
						"
					>
						<PropertyField
							id="loadpointMinCurrent"
							v-model="values.minCurrent"
							type="Float"
							unit="A"
							size="w-25 w-min-200"
							class="me-2"
							required
						/>
					</FormRow>

					<FormRow
						id="loadpointMaxCurrent"
						label="Maximum current"
						class="col-sm-6 mb-sm-0"
						:help="
							values.maxCurrent < values.minCurrent
								? 'Must be greater than minimum current.'
								: null
						"
					>
						<PropertyField
							id="loadpointMaxCurrent"
							v-model="values.maxCurrent"
							type="Float"
							unit="A"
							size="w-25 w-min-200"
							class="me-2"
							required
						/>
					</FormRow>
				</div>

				<FormRow
					v-if="showCircuit"
					id="loadpointParamCircuit"
					label="Circuit"
					help="Select load management circuit for this charge point."
				>
					<PropertyField
						id="loadpointParamCircuit"
						v-model="values.circuit"
						type="String"
						class="me-2"
						:valid-values="circuitOptions"
						required
					/>
				</FormRow>

				<h6>Vehicles</h6>

				<div v-if="vehicleOptions.length">
					<FormRow
						id="loadpointParamVehicle"
						label="Default vehicle"
						:help="
							values.vehicle
								? 'Always assume this vehicle is charging here. Auto-detection disabled. Manual override is possible.'
								: 'Automatically selects the most plausible vehicle. Manual override is possible.'
						"
					>
						<PropertyField
							id="loadpointParamVehicle"
							v-model="values.vehicle"
							type="String"
							class="me-2"
							:valid-values="allVehicleOptions"
							required
						/>
					</FormRow>

					<FormRow
						id="loadpointPollMode"
						label="Update behaviour"
						:help="
							values.soc.poll.mode === 'charging'
								? 'Only request vehicle status updates when charging.'
								: values.soc.poll.mode === 'connected'
									? 'Update vehicle status in regular intervals when connected.'
									: values.soc.poll.mode === 'always'
										? 'Always request status updates in regular intervals.'
										: null
						"
					>
						<SelectGroup
							id="loadpointPollMode"
							v-model="values.soc.poll.mode"
							class="w-100"
							:options="[
								{ value: 'charging', name: 'charging' },
								{ value: 'connected', name: 'connected' },
								{ value: 'always', name: 'always' },
							]"
							transparent
						/>
					</FormRow>
					<FormRow
						v-if="values.soc.poll.mode !== 'charging'"
						id="loadpointPollInterval"
						class="ms-3 mb-5"
						label="Udate interval"
						help="Time between vehicle API updates. Short intervals may drain the vehicle battery."
					>
						<PropertyField
							id="loadpointPollInterval"
							v-model="values.soc.poll.interval"
							type="Duration"
							unit="minute"
							size="w-25 w-min-200"
							class="me-2"
							required
						/>
					</FormRow>

					<div class="d-flex mb-4">
						<input
							id="loadpointEstimate"
							v-model="values.soc.estimate"
							class="form-check-input"
							type="checkbox"
						/>
						<label class="form-check-label ms-2" for="loadpointEstimate">
							Interpolate charge level between API updates
						</label>
					</div>
				</div>
				<div v-else>
					<p class="text-muted">No vehicles are configured.</p>
				</div>
			</div>

			<div v-if="values.charger" class="mt-5 mb-4 d-flex justify-content-between">
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
import SelectGroup from "../SelectGroup.vue";
import api from "../../api";
import GenericModal from "../GenericModal.vue";
import deepClone from "../../utils/deepClone";
import deepEqual from "../../utils/deepEqual";
import formatter, { POWER_UNIT } from "../../mixins/formatter";
import EditIcon from "../MaterialIcon/Edit.vue";

const nsPerMin = 60 * 1e9;

const defaultValues = {
	title: "",
	phases: 3,
	minCurrent: 6,
	maxCurrent: 16,
	priority: 0,
	mode: "",
	thresholds: {
		enable: { delay: 1 * nsPerMin, threshold: 0 },
		disable: { delay: 3 * nsPerMin, threshold: 0 },
	},
	soc: {
		poll: { mode: "charging", interval: 60 * nsPerMin },
		estimate: false,
	},
	vehicle: "",
	charger: "",
	circuit: "",
	meter: "",
};

const defaultThresholds = {
	enable: { delay: 1 * nsPerMin, threshold: 0 },
	disable: { delay: 3 * nsPerMin, threshold: 0 },
};

export default {
	name: "LoadpointModal",
	components: { FormRow, PropertyField, GenericModal, SelectGroup, EditIcon },
	mixins: [formatter],
	props: {
		id: Number,
		name: String,
		vehicleOptions: { type: Array, default: () => [] },
		loadpointCount: Number,
		fade: String,
		chargers: { type: Array, default: () => [] },
		meters: { type: Array, default: () => [] },
		circuits: { type: Array, default: () => [] },
	},
	emits: ["updated", "openMeterModal", "openChargerModal", "close", "opened"],
	data() {
		return {
			isModalVisible: false,
			saving: false,
			selectedType: null,
			values: deepClone(defaultValues),
			chargerPower: "11kw",
			solarMode: "summer",
			tab: "solar",
			powerUnit: POWER_UNIT,
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
		chargerTitle() {
			const name = this.values.charger;
			if (!name) return "";
			const charger = this.chargers.find((c) => c.name === name);
			const title = charger?.config?.template || "unknown";
			return `${title} [${name}]`;
		},
		meterTitle() {
			const name = this.values.meter;
			if (!name) return "";
			const meter = this.meters.find((m) => m.name === name);
			const title = meter?.config?.template || "unknown";
			return `${title} [${name}]`;
		},
		isDeletable() {
			return !this.isNew;
		},
		showPriority() {
			return this.priorityOptions.length > 1;
		},
		priorityOptions() {
			const maxPriority = this.loadpointCount + (this.isNew ? 1 : 0);
			const result = Array.from({ length: maxPriority + 1 }, (_, i) => ({
				value: i,
				name: `${i}`,
			}));
			return result;
		},
		showCircuit() {
			return this.circuits.length > 0;
		},
		circuitOptions() {
			const options = this.circuits.map((c) => ({
				key: c.name,
				name: `${c.config?.title || ""} [${c.name}]`.trim(),
			}));
			return [{ key: "", name: "unassigned" }, ...options];
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
		chargerPower(value) {
			if (value === "11kw") {
				this.values.minCurrent = 6;
				this.values.maxCurrent = 16;
			} else if (value === "22kw") {
				this.values.minCurrent = 6;
				this.values.maxCurrent = 32;
			}
		},
		solarMode(value) {
			if (value === "default") {
				this.values.thresholds = deepClone(defaultThresholds);
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
				this.values = deepClone(res.data.result);
				this.updateChargerPower();
				this.updateSolarMode();
			} catch (e) {
				console.error(e);
			}
		},
		async update() {
			this.saving = true;
			try {
				const values = deepClone(this.values);
				await api.put(`config/loadpoints/${this.id}`, values);
				this.$emit("updated");
				this.close();
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
				this.close();
			} catch (e) {
				console.error(e);
				alert("delete failed");
			}
		},
		async create() {
			this.saving = true;
			try {
				const values = this.transformBeforeSave(this.values);
				await api.post("config/loadpoints", values);
				this.$emit("updated");
				this.close();
			} catch (e) {
				console.error(e);
				alert("create failed");
			}
			this.saving = false;
		},
		open() {
			this.isModalVisible = true;
		},
		opened() {
			this.$emit("opened");
		},
		close() {
			this.$emit("close");
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
		updateChargerPower() {
			const { minCurrent, maxCurrent } = this.values;
			if (minCurrent === 6 && maxCurrent === 16) {
				this.chargerPower = "11kw";
			} else if (minCurrent === 6 && maxCurrent === 32) {
				this.chargerPower = "22kw";
			} else {
				this.chargerPower = "other";
			}
		},
		updateSolarMode() {
			const { thresholds } = this.values;
			if (deepEqual(thresholds, defaultThresholds)) {
				this.solarMode = "default";
			} else {
				this.solarMode = "custom";
			}
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
h6 {
	margin-top: 4rem;
}
</style>
