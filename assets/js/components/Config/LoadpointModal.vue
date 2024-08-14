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
			<FormRow id="loadpointParamCharger" label="Charger">
				<div class="d-flex">
					<PropertyField
						id="loadpointParamCharger"
						v-model="values.charger"
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
			<div v-if="values.charger || !isNew">
				<FormRow
					v-if="values.meter"
					class="offset-1 mb-5"
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
				<FormRow id="loadpointMode" label="Mode" help="Mode when connecting the vehicle.">
					<PropertyField
						id="loadpointMode"
						v-model="values.mode"
						type="String"
						class="w-100"
						required
						:valid-values="[
							{ value: '', name: 'Keep last selection' },
							{ value: 'off', name: $t('main.mode.off') },
							{ value: 'pv', name: $t('main.mode.pv') },
							{ value: 'minpv', name: $t('main.mode.minpv') },
							{ value: 'now', name: $t('main.mode.now') },
						]"
					/>
				</FormRow>
				<FormRow
					id="loadpointParamPhases"
					label="Phases"
					:help="
						values.phases === 0
							? `Select this if your charger supports automatic phase switching.`
							: `Used for calculating the minimum charge power in solar mode.`
					"
				>
					<SelectGroup
						id="loadpointParamPhases"
						v-model="values.phases"
						class="w-100"
						:options="[
							{ value: 1, name: '1-phase' },
							{ value: 3, name: '3-phase' },
							{ value: 0, name: 'automatic' },
						]"
					/>
				</FormRow>

				<FormRow
					id="chargerPower"
					label="Charger type"
					help="Defines the minimum and maximum current used for charging."
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
					/>
				</FormRow>

				<div v-if="chargerPower === 'other'" class="row offset-1 mb-5">
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
					id="loadpointSolarMode"
					label="Solar behaviour"
					:help="
						solarMode === 'summer'
							? `Only use solar surplus. Minimize grid use. Faster start (${fmtDuration(values.thresholds.enable.delay * 60)}) and stop (${fmtDuration(values.thresholds.disable.delay * 60)}).`
							: solarMode === 'winter'
								? `Use more solar surplus. Allow grid use. Slower start (${fmtDuration(values.thresholds.enable.delay * 60)}) and stop (${fmtDuration(values.thresholds.disable.delay * 60)}).`
								: null
					"
				>
					<SelectGroup
						id="loadpointSolarMode"
						v-model="solarMode"
						class="w-100"
						:options="[
							{ name: 'summer', value: 'summer' },
							{ name: 'winter', value: 'winter' },
							{ name: 'custom', value: 'custom' },
						]"
					/>
				</FormRow>

				<div v-show="solarMode === 'custom'" class="offset-1 mb-4">
					<div class="d-flex flex-wrap flex-sm-nowrap gap-4 ps-1 ms-2">
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
								unit="min"
								size="w-25 w-min-200"
								required
							/>
						</FormRow>
					</div>
					<div class="form-text evcc-gray ps-1 ms-2">
						{{
							values.thresholds.enable.threshold === 0
								? `Start when minimum charge power surplus is available for ${fmtDuration(values.thresholds.enable.delay * 60)}.`
								: values.thresholds.enable.threshold < 0
									? `Start charging, when ${fmtKw(-1 * values.thresholds.enable.threshold, false)} surplus is available for ${fmtDuration(values.thresholds.enable.delay * 60)}.`
									: `Please use a negative value.`
						}}
					</div>
				</div>

				<div v-show="solarMode === 'custom'" class="offset-1 mb-5">
					<div class="d-flex flex-wrap flex-sm-nowrap gap-4 ps-1 ms-2">
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
								unit="min"
								size="w-25 w-min-200"
								required
							/>
						</FormRow>
					</div>
					<div class="form-text evcc-gray ps-1 ms-2">
						{{
							values.thresholds.disable.threshold === 0
								? `Stop when minimum charge power can't be satisfied for ${fmtDuration(values.thresholds.disable.delay * 60)}.`
								: values.thresholds.disable.threshold > 0
									? `Stop charging, when more than ${fmtKw(values.thresholds.disable.threshold, false)} is used from the grid for ${fmtDuration(values.thresholds.disable.delay * 60)}.`
									: `Please use a positive value.`
						}}
					</div>
				</div>

				<FormRow
					v-if="showPriority"
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

				<div v-if="vehicleOptions.length">
					<hr class="my-4" />

					<FormRow
						id="loadpointParamVehicle"
						label="Vehicle"
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
						label="Vehicle updates"
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
						/>
					</FormRow>
					<FormRow
						v-if="values.soc.poll.mode !== 'charging'"
						class="offset-1"
						id="loadpointPollInterval"
						label="Vehicle update interval"
						help="Time between status updates. Short intervals may drain the vehicle battery."
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

					<div class="d-flex offset-1 mb-5">
						<input
							class="form-check-input"
							id="loadpointEstimate"
							type="checkbox"
							v-model="values.soc.estimate"
						/>
						<label class="form-check-label ms-2" for="loadpointEstimate">
							Interpolate charge level between API updates
						</label>
					</div>
				</div>
			</div>

			<div class="mb-4 d-flex justify-content-between">
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
import formatter from "../../mixins/formatter";
import EditIcon from "../MaterialIcon/Edit.vue";

const nsPerMin = 60 * 1e9;
const wPerKw = 1e3;

const defaultValues = {
	title: "",
	phases: 3,
	minCurrent: 6,
	maxCurrent: 16,
	priority: 0,
	mode: "",
	thresholds: {
		enable: { delay: 1, threshold: 0 },
		disable: { delay: 3, threshold: 0 },
	},
	soc: {
		poll: { mode: "charging", interval: 60 },
		estimate: false,
	},
	vehicle: "",
	charger: "",
	meter: "",
};

const summerThresholds = {
	enable: { delay: 1, threshold: 0 },
	disable: { delay: 3, threshold: 0 },
};
const winterThresholds = {
	enable: { delay: 1, threshold: -2000 },
	disable: { delay: 15, threshold: 2000 },
};

export default {
	name: "LoadpointModal",
	components: { FormRow, PropertyField, GenericModal, SelectGroup, EditIcon },
	props: {
		id: Number,
		name: String,
		vehicleOptions: { type: Array, default: () => [] },
		loadpointCount: Number,
		fade: String,
	},
	mixins: [formatter],
	emits: ["updated", "openMeterModal", "openChargerModal", "close", "opened"],
	data() {
		return {
			isModalVisible: false,
			saving: false,
			selectedType: null,
			values: deepClone(defaultValues),
			chargerPower: "11kw",
			solarMode: "summer",
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
		showPriority() {
			let count = this.loadpointCount;
			if (this.isNew) count++;
			return count > 1;
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
			if (value === "summer") {
				this.values.thresholds = deepClone(summerThresholds);
			} else if (value === "winter") {
				this.values.thresholds = deepClone(winterThresholds);
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
				this.updateChargerPower();
				this.updateSolarMode();
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
			if (deepEqual(thresholds, summerThresholds)) {
				this.solarMode = "summer";
			} else if (deepEqual(thresholds, winterThresholds)) {
				this.solarMode = "winter";
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
.btn-cancel {
	margin-left: -0.75rem;
}
h6 {
	margin-top: 4rem;
}
</style>
import formatter from "assets/js/mixins/formatter";import deepClone from
"../../utils/deepClone";import deepEqual from "assets/js/utils/deepEqual";import deepEqual from
"assets/js/utils/deepEqual";
