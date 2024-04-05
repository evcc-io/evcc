<template>
	<Teleport to="body">
		<div
			id="loadpointModal"
			ref="modal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
			data-testid="loadpoint-modal"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ modalTitle }}
						</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<form ref="form" class="container mx-0 px-0">
							<FormRow
								id="loadpointParamTitle"
								label="Title"
								example="Garage, Carport, etc."
							>
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
									<button class="btn btn-link btn-sm evcc-default-text">
										<shopicon-regular-adjust></shopicon-regular-adjust>
									</button>
								</div>
							</FormRow>
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
									<button class="btn btn-link btn-sm evcc-default-text">
										<shopicon-regular-adjust></shopicon-regular-adjust>
									</button>
								</div>
							</FormRow>

							<h4 class="mb-3 mt-5 text-evcc">Basics</h4>

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
											{ key: 2, name: '2-phase' },
											{ key: 3, name: '3-phase' },
										]"
										required
									/>
								</FormRow>
							</div>

							<FormRow id="loadpointParamMaxCurrent" label="Current limits">
								<CurrentRange
									v-model:min="values.minCurrent"
									v-model:max="values.maxCurrent"
								/>
							</FormRow>

							<h4 class="mb-3 mt-5 text-evcc">Solar behaviour</h4>

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
								<FormRow
									id="loadpointEnableDelay"
									label="Enable Delay"
									class="col-sm-6"
								>
									<PropertyField
										id="loadpointEnableDelay"
										v-model="values.thresholds.enable.delay"
										type="Duration"
										unit="min"
										:scale="minuteScale"
										size="w-25 w-min-200"
										class="me-2"
										required
									/>
								</FormRow>
								<FormRow
									id="loadpointDisableDelay"
									label="Disable Delay"
									class="col-sm-6"
								>
									<PropertyField
										id="loadpointDisableDelay"
										v-model="values.thresholds.disable.delay"
										type="Duration"
										unit="min"
										:scale="minuteScale"
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
									class="col-sm-6"
								>
									<PropertyField
										id="loadpointEnableThreshold"
										v-model="values.thresholds.enable.threshold"
										type="Number"
										unit="kW"
										size="w-25 w-min-200"
										class="me-2"
										required
									/>
								</FormRow>

								<FormRow
									id="loadpointDisableThreshold"
									label="Disable Threshold"
									class="col-sm-6"
								>
									<PropertyField
										id="loadpointDisableThreshold"
										v-model="values.thresholds.disable.threshold"
										type="Number"
										unit="kW"
										size="w-25 w-min-200"
										class="me-2"
										required
									/>
								</FormRow>
							</div>

							<h4 class="mb-3 mt-4 text-evcc">Vehicle</h4>

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
											{ key: 0, name: 'when charging' },
											{ key: 1, name: 'when connected' },
											{ key: 2, name: 'always' },
										]"
										required
									/>
								</FormRow>
								<FormRow
									v-if="values.soc.poll.mode > 0"
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
										:scale="minuteScale"
										size="w-25 w-min-200"
										class="me-2"
										required
									/>
								</FormRow>
							</div>

							<FormRow
								id="loadpointEstimate"
								label="Estimate charge level"
								help="If enabled, the charge level will be interpolated based on charge power and duration."
							>
								<PropertyField
									id="loadpointEstimate"
									v-model="values.soc.estimate"
									type="Boolean"
									class="me-2"
									:valid-values="[
										{ key: false, name: 'no (only use vehicle data)' },
										{ key: true, name: 'yes (interpolate between updates)' },
									]"
									required
								/>
							</FormRow>

							<div class="mt-5 mb-4 d-flex justify-content-between">
								<button
									type="button"
									class="btn btn-link text-muted"
									data-bs-dismiss="modal"
								>
									{{ $t("config.loadpoint.cancel") }}
								</button>
								<button
									type="submit"
									class="btn btn-primary"
									:disabled="saving"
									@click.prevent="isNew ? create() : update()"
								>
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
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import CurrentRange from "./CurrentRange.vue";
import api from "../../api";

const defaultValues = {
	title: "",
	phases: 0,
	minCurrent: 0,
	maxCurrent: 0,
	priority: 0,
	mode: "",
	thresholds: {
		enable: { delay: 0, threshold: 0 },
		disable: { delay: 0, threshold: 0 },
	},
	soc: {
		poll: { mode: 0, interval: 0 },
		estimate: false,
	},
	defaultVehicle: "",
	charger: "",
	meter: "",
};

export default {
	name: "LoadpointModal",
	components: { FormRow, PropertyField, CurrentRange },
	props: {
		id: Number,
		name: String,
		vehicleOptions: { type: Array, default: () => [] },
	},
	emits: ["added", "updated", "removed"],
	data() {
		return {
			isModalVisible: false,
			saving: false,
			selectedType: null,
			values: defaultValues,
		};
	},
	computed: {
		minuteScale() {
			return 1 / 60 / 1e9;
		},
		kWScale() {
			return 1 / 1e3;
		},
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
				this.reset();
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
		},
	},
	mounted() {
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hide.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal?.removeEventListener("hide.bs.modal", this.modalInvisible);
	},
	methods: {
		reset() {
			this.values = defaultValues;
		},
		async loadConfiguration() {
			try {
				this.values = (await api.get(`config/loadpoints/${this.id}`)).data.result;
			} catch (e) {
				console.error(e);
			}
		},
		async update() {
			this.saving = true;
			try {
				await api.put(`config/loadpoints/${this.id}`, this.values);
				this.$emit("updated");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("update failed");
			}
			this.saving = false;
		},
		modalVisible() {
			this.isModalVisible = true;
		},
		modalInvisible() {
			this.isModalVisible = false;
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
</style>
