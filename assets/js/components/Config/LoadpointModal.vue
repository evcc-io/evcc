<template>
	<GenericModal
		id="loadpointModal"
		ref="modal"
		:title="modalTitle"
		data-testid="loadpoint-modal"
		:fade="fade"
		@open="onOpen"
		@opened="onOpened"
		@close="onClose"
	>
		<div v-if="!loadpointType" class="d-flex flex-column gap-4">
			<NewDeviceButton
				v-for="t in typeChoices"
				:key="t"
				:title="$t(`config.loadpoint.option.${t}`)"
				class="addButton"
				@click="selectType(t)"
			/>
		</div>
		<form
			v-else
			ref="form"
			class="container mx-0 px-0"
			@submit.prevent="isNew ? create() : update()"
		>
			<FormRow
				id="loadpointParamTitle"
				:label="$t('config.loadpoint.titleLabel')"
				:example="
					loadpointType ? $t(`config.loadpoint.titleExample.${loadpointType}`) : undefined
				"
			>
				<PropertyField
					id="loadpointParamTitle"
					v-model="values.title"
					type="String"
					class="me-2"
					required
				/>
			</FormRow>
			<FormRow
				v-if="charger || !isNew"
				id="loadpointParamCharger"
				:label="$t(`config.loadpoint.chargerLabel.${loadpointType}`)"
				:error="!charger ? $t(`config.loadpoint.chargerError.${loadpointType}`) : undefined"
			>
				<div class="d-flex">
					<PropertyField
						id="loadpointParamCharger"
						:modelValue="chargerTitle"
						type="String"
						class="me-2 flex-grow-1"
						readonly
						required
						:invalid="!charger || hasDeviceError('charger', values.charger)"
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
					{{ addChargerLabel }}
				</button>
			</div>
			<div v-if="charger || !isNew">
				<FormRow
					v-if="values.meter"
					id="loadpointParamMeter"
					class="mb-6"
					:label="$t('config.loadpoint.energyMeterLabel')"
					:help="$t('config.loadpoint.energyMeterHelp')"
				>
					<div class="d-flex">
						<PropertyField
							id="loadpointParamMeter"
							:modelValue="meterTitle"
							type="String"
							class="me-2 flex-grow-1"
							required
							readonly
							:invalid="hasDeviceError('meter', values.meter)"
							@click="editMeter"
						/>
						<button
							class="btn btn-link btn-sm evcc-default-text"
							type="button"
							tabindex="0"
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
						tabindex="0"
						@click="editMeter"
					>
						{{ $t(`config.loadpoint.addMeter`) }}
					</button>
				</p>
			</div>

			<div v-if="values.charger || !isNew">
				<h6>{{ $t("config.loadpoint.chargingTitle") }}</h6>

				<FormRow
					id="loadpointMode"
					:label="$t('config.loadpoint.defaultModeLabel')"
					:help="
						values.defaultMode === ''
							? $t(`config.loadpoint.defaultModeHelpKeep`)
							: $t(`config.loadpoint.defaultModeHelp.${loadpointType}`)
					"
				>
					<PropertyField
						id="loadpointMode"
						v-model="values.defaultMode"
						type="Choice"
						class="w-100"
						required
						:choice="[
							{ key: '', name: '---' },
							{ key: 'off', name: $t('main.mode.off') },
							{ key: 'pv', name: $t('main.mode.pv') },
							{ key: 'minpv', name: $t('main.mode.minpv') },
							{ key: 'now', name: $t('main.mode.now') },
						]"
					/>
				</FormRow>

				<FormRow
					id="loadpointSolarMode"
					:label="$t('config.loadpoint.solarBehaviorLabel')"
					:help="
						solarMode === 'default'
							? $t('config.loadpoint.solarBehaviorDefaultHelp', {
									enableDelay: fmtDurationNs(
										values.thresholds.enable.delay,
										true,
										'm'
									),
									disableDelay: fmtDurationNs(
										values.thresholds.disable.delay,
										true,
										'm'
									),
								})
							: $t('config.loadpoint.solarBehaviorCustomHelp')
					"
				>
					<SelectGroup
						id="loadpointSolarMode"
						v-model="solarMode"
						class="w-100"
						:options="[
							{ name: $t('config.loadpoint.solarModeMaximum'), value: 'default' },
							{ name: $t('config.loadpoint.solarModeCustom'), value: 'custom' },
						]"
						transparent
						equal-width
					/>
				</FormRow>

				<div v-show="solarMode === 'custom'" class="ms-3 mb-5">
					<div class="mb-4">
						<div class="d-flex flex-wrap flex-sm-nowrap gap-4">
							<FormRow
								id="loadpointEnableThreshold"
								:label="$t('config.loadpoint.thresholdEnableLabel')"
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
								:label="$t('config.loadpoint.thresholdEnableDelayLabel')"
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
									? $t("config.loadpoint.thresholdEnableHelpZero", {
											delay: fmtDurationNs(
												values.thresholds.enable.delay,
												true,
												"m"
											),
										})
									: values.thresholds.enable.threshold < 0
										? $t("config.loadpoint.thresholdEnableHelpNegative", {
												surplus: fmtW(
													-1 * values.thresholds.enable.threshold,
													powerUnit.AUTO
												),
												delay: fmtDurationNs(
													values.thresholds.enable.delay,
													true,
													"m"
												),
											})
										: $t("config.loadpoint.thresholdEnableHelpInvalid")
							}}
						</div>
					</div>

					<div>
						<div class="d-flex flex-wrap flex-sm-nowrap gap-4">
							<FormRow
								id="loadpointDisableThreshold"
								:label="$t('config.loadpoint.thresholdDisableLabel')"
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
								:label="$t('config.loadpoint.thresholdDisableDelayLabel')"
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
									? $t("config.loadpoint.thresholdDisableHelpZero", {
											delay: fmtDurationNs(
												values.thresholds.disable.delay,
												true,
												"m"
											),
										})
									: values.thresholds.disable.threshold > 0
										? $t("config.loadpoint.thresholdDisableHelpPositive", {
												power: fmtW(
													values.thresholds.disable.threshold,
													powerUnit.AUTO
												),
												delay: fmtDurationNs(
													values.thresholds.disable.delay,
													true,
													"m"
												),
											})
										: $t("config.loadpoint.thresholdDisableHelpInvalid")
							}}
						</div>
					</div>
				</div>

				<FormRow
					v-if="showPriority"
					id="loadpointParamPriority"
					:label="$t('config.loadpoint.priorityLabel')"
					:help="$t('config.loadpoint.priorityHelp')"
				>
					<PropertyField
						id="loadpointParamPriority"
						v-model="values.priority"
						type="Choice"
						size="w-100"
						class="me-2"
						:choice="priorityOptions"
						required
					/>
				</FormRow>

				<h6>
					{{ $t("config.loadpoint.electricalTitle") }}
					<small class="text-muted">{{
						$t("config.loadpoint.electricalSubtitle")
					}}</small>
				</h6>

				<FormRow
					id="chargerPower"
					:label="$t('config.loadpoint.chargerTypeLabel')"
					:help="
						chargerPower === '11kw'
							? $t('config.loadpoint.chargerPower11kwHelp')
							: chargerPower === '22kw'
								? $t('config.loadpoint.chargerPower22kwHelp')
								: $t('config.loadpoint.chargerPowerCustomHelp')
					"
				>
					<SelectGroup
						id="chargerPower"
						v-model="chargerPower"
						class="w-100"
						:options="[
							{ name: $t('config.loadpoint.chargerPower11kw'), value: '11kw' },
							{ name: $t('config.loadpoint.chargerPower22kw'), value: '22kw' },
							{ name: $t('config.loadpoint.chargerPowerCustom'), value: 'other' },
						]"
						transparent
					/>
				</FormRow>

				<div v-if="chargerPower === 'other'" class="row ms-3 mb-5">
					<FormRow
						id="loadpointMinCurrent"
						:label="$t('config.loadpoint.minCurrentLabel')"
						class="col-sm-6 mb-sm-0"
						:help="
							values.minCurrent < 6
								? $t('config.loadpoint.minCurrentHelp')
								: undefined
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
						:label="$t('config.loadpoint.maxCurrentLabel')"
						class="col-sm-6 mb-sm-0"
						:help="
							values.maxCurrent < values.minCurrent
								? $t('config.loadpoint.maxCurrentHelp')
								: undefined
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

				<template v-if="!chargerIsSinglePhase">
					<FormRow
						v-if="chargerSupports1p3p"
						id="loadpointParamPhases"
						:label="$t('config.loadpoint.phasesAutomatic')"
						:help="$t('config.loadpoint.phasesAutomaticHelp')"
					>
					</FormRow>
					<FormRow
						v-else
						id="loadpointParamPhases"
						:label="$t('config.loadpoint.phasesLabel')"
						:help="$t('config.loadpoint.phasesHelp')"
					>
						<SelectGroup
							id="loadpointParamPhases"
							v-model="values.phasesConfigured"
							class="w-100"
							:options="phasesOptions"
							transparent
							equal-width
						/>
					</FormRow>
				</template>

				<FormRow
					v-if="showCircuit"
					id="loadpointParamCircuit"
					:label="$t('config.loadpoint.circuitLabel')"
					:help="$t('config.loadpoint.circuitHelp')"
				>
					<PropertyField
						id="loadpointParamCircuit"
						v-model="values.circuit"
						type="Choice"
						class="me-2"
						:choice="circuitOptions"
						required
					/>
				</FormRow>

				<div v-if="!chargerIsIntegratedDevice">
					<h6>{{ $t("config.loadpoint.vehiclesTitle") }}</h6>

					<div v-if="vehicleOptions.length">
						<FormRow
							id="loadpointParamVehicle"
							:label="$t('config.loadpoint.vehicleLabel')"
							:help="
								values.vehicle
									? $t('config.loadpoint.vehicleHelpDefault')
									: $t('config.loadpoint.vehicleHelpAutoDetection')
							"
						>
							<PropertyField
								id="loadpointParamVehicle"
								v-model="values.vehicle"
								type="Choice"
								class="me-2"
								:choice="allVehicleOptions"
								required
							/>
						</FormRow>

						<FormRow
							id="loadpointPollMode"
							:label="$t('config.loadpoint.pollModeLabel')"
							:help="
								values.soc.poll.mode === 'charging'
									? $t('config.loadpoint.pollModeChargingHelp')
									: values.soc.poll.mode === 'connected'
										? $t('config.loadpoint.pollModeConnectedHelp')
										: values.soc.poll.mode === 'always'
											? $t('config.loadpoint.pollModeAlwaysHelp')
											: undefined
							"
						>
							<SelectGroup
								id="loadpointPollMode"
								v-model="values.soc.poll.mode"
								class="w-100"
								:options="[
									{
										value: 'charging',
										name: $t('config.loadpoint.pollModeCharging'),
									},
									{
										value: 'connected',
										name: $t('config.loadpoint.pollModeConnected'),
									},
									{
										value: 'always',
										name: $t('config.loadpoint.pollModeAlways'),
									},
								]"
								transparent
							/>
						</FormRow>
						<FormRow
							v-if="values.soc.poll.mode !== 'charging'"
							id="loadpointPollInterval"
							class="ms-3 mb-5"
							:label="$t('config.loadpoint.pollIntervalLabel')"
							:help="$t('config.loadpoint.pollIntervalHelp')"
							:danger="$t('config.loadpoint.pollIntervalDanger')"
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

						<div>
							<div class="d-flex mb-4">
								<input
									id="loadpointEstimate"
									v-model="values.soc.estimate"
									class="form-check-input"
									type="checkbox"
								/>
								<label class="form-check-label ms-2" for="loadpointEstimate">
									{{ $t("config.loadpoint.estimateLabel") }}
								</label>
							</div>
						</div>
					</div>
					<div v-else>
						<p class="text-muted">{{ $t("config.loadpoint.noVehicles") }}</p>
					</div>
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

<script lang="ts">
import type { PropType } from "vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import SelectGroup from "../Helper/SelectGroup.vue";
import api from "@/api";
import GenericModal from "../Helper/GenericModal.vue";
import deepClone from "@/utils/deepClone";
import deepEqual from "@/utils/deepEqual";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import EditIcon from "../MaterialIcon/Edit.vue";
import NewDeviceButton from "./NewDeviceButton.vue";
import { handleError, customChargerName } from "./DeviceModal";
import {
	LOADPOINT_TYPE,
	type DeviceType,
	type LoadpointType,
	type ConfigCharger,
	type ConfigMeter,
	type ConfigCircuit,
	type ConfigLoadpoint,
} from "@/types/evcc";

const nsPerMin = 60 * 1e9;

const defaultValues = {
	id: undefined,
	title: "",
	phasesConfigured: 3,
	minCurrent: 6,
	maxCurrent: 16,
	priority: 0,
	defaultMode: "",
	thresholds: {
		enable: { delay: 1 * nsPerMin, threshold: 0 },
		disable: { delay: 3 * nsPerMin, threshold: 0 },
	},
	soc: {
		poll: { mode: "charging", interval: 60 * nsPerMin },
		estimate: true,
	},
	vehicle: "",
	charger: "",
	circuit: "",
	meter: "",
} as ConfigLoadpoint;

const defaultThresholds = {
	enable: { delay: 1 * nsPerMin, threshold: 0 },
	disable: { delay: 3 * nsPerMin, threshold: 0 },
};

export default {
	name: "LoadpointModal",
	components: { FormRow, PropertyField, GenericModal, SelectGroup, EditIcon, NewDeviceButton },
	mixins: [formatter],
	props: {
		id: Number,
		name: String,
		vehicleOptions: { type: Array, default: () => [] },
		loadpointCount: { type: Number, default: 0 },
		fade: String,
		chargers: { type: Array as PropType<ConfigCharger[]>, default: () => [] },
		chargerValues: { type: Object, default: () => {} },
		meters: { type: Array as PropType<ConfigMeter[]>, default: () => [] },
		circuits: { type: Array as PropType<ConfigCircuit[]>, default: () => [] },
		hasDeviceError: {
			type: Function as PropType<(type: DeviceType, name: string) => boolean>,
			default: () => false,
		},
	},
	emits: ["updated", "openMeterModal", "openChargerModal", "opened"],
	data() {
		return {
			isModalVisible: false,
			saving: false,
			selectedType: null as LoadpointType | null,
			values: deepClone(defaultValues) as ConfigLoadpoint,
			chargerPower: "11kw",
			solarMode: "default",
			tab: "solar",
			powerUnit: POWER_UNIT,
		};
	},
	computed: {
		modalTitle() {
			if (this.isNew) {
				return this.$t(`config.loadpoint.titleAdd.${this.loadpointType || "unknown"}`);
			}
			return this.$t(`config.loadpoint.titleEdit.${this.loadpointType || "unknown"}`);
		},
		isNew() {
			return this.id === undefined;
		},
		charger() {
			return this.chargers.find((c) => c.name === this.values.charger);
		},
		chargerType() {
			if (!this.charger) {
				return null;
			}
			return this.chargerIsHeating ? LOADPOINT_TYPE.HEATING : LOADPOINT_TYPE.CHARGING;
		},
		chargerTitle() {
			if (!this.charger) return "";
			const title =
				this.charger.deviceProduct ||
				this.charger.config?.template ||
				this.$t(customChargerName(this.charger.type, this.chargerIsHeating));
			return `${title} [${this.values.charger}]`;
		},
		chargerStatus() {
			if (!this.chargerValues || !this.values.charger) {
				return {};
			}
			return this.chargerValues[this.values.charger] || {};
		},
		chargerSupports1p3p() {
			return this.chargerStatus.phases1p3p?.value || false;
		},
		chargerIsSinglePhase() {
			return this.chargerStatus.singlePhase?.value || false;
		},
		chargerIsIntegratedDevice() {
			return this.chargerStatus.integratedDevice?.value || false;
		},
		chargerIsHeating() {
			return this.chargerStatus.heating?.value === true;
		},
		meterTitle() {
			const name = this.values.meter;
			if (!name) return "";
			const meter = this.meters.find((m) => m.name === name);
			const title =
				meter?.deviceProduct ||
				meter?.config?.template ||
				this.$t("config.general.customOption");
			return `${title} [${name}]`;
		},
		isDeletable() {
			return !this.isNew;
		},
		showPriority() {
			return this.isNew ? this.loadpointCount > 0 : this.loadpointCount > 1;
		},
		priorityOptions() {
			const result = Array.from({ length: 11 }, (_, i) => ({ key: i, name: `${i}` })) as {
				key?: number;
				name: string;
			}[];
			result[0].name = "0 (default)";
			result[0].key = undefined;
			result[10].name = "10 (highest)";
			return result;
		},
		phasesOptions() {
			return [
				{ value: "1", name: this.$t("config.loadpoint.phases1p") },
				{ value: "3", name: this.$t("config.loadpoint.phases3p") },
			];
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
				{ key: "", name: this.$t("config.loadpoint.vehicleAutoDetection") },
				{ key: null, name: null },
				...this.vehicleOptions,
			];
		},
		typeChoices(): LoadpointType[] {
			return Object.values(LOADPOINT_TYPE);
		},
		loadpointType(): LoadpointType | null {
			return this.selectedType ?? this.chargerType;
		},
		addChargerLabel() {
			if (this.loadpointType) {
				return this.$t(`config.loadpoint.addCharger.${this.loadpointType}`);
			}
			return "";
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
		chargerSupports1p3p() {
			this.updatePhases();
		},
		chargerIsSinglePhase() {
			this.updatePhases();
		},
	},
	methods: {
		reset() {
			this.selectedType = null;
			this.values = deepClone(defaultValues);
			this.updatePhases();
		},
		async loadConfiguration() {
			try {
				const res = await api.get(`config/loadpoints/${this.id}`);
				this.values = deepClone(res.data);
				this.updateChargerPower();
				this.updateSolarMode();
				this.updatePhases();
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
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await api.delete(`config/loadpoints/${this.id}`);
				this.$emit("updated");
				(this.$refs["modal"] as any).close();
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
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "create failed");
			}
			this.saving = false;
		},
		onOpen() {
			this.isModalVisible = true;
		},
		onOpened() {
			this.$emit("opened");
		},
		onClose() {
			this.isModalVisible = false;
		},
		editCharger() {
			this.$emit("openChargerModal", this.values.charger, this.loadpointType);
		},
		editMeter() {
			this.$emit("openMeterModal", this.values.meter);
		},
		// called externally
		setMeter(meter: string) {
			this.values.meter = meter;
		},
		// called externally
		setCharger(charger: string) {
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
		updatePhases() {
			const { phasesConfigured } = this.values;
			if (this.chargerIsSinglePhase) {
				this.values.phasesConfigured = 1;
				return;
			}
			if (this.chargerSupports1p3p && this.isNew) {
				this.values.phasesConfigured = 0; // automatic
				return;
			}
			if (!this.chargerSupports1p3p && phasesConfigured === 0) {
				this.values.phasesConfigured = 3; // no automatic switching, default to 3-phase
				return;
			}
		},
		selectType(type: LoadpointType) {
			this.selectedType = type;
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
