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
							<hr class="mt-3 mb-4" />
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

							<div class="row">
								<FormRow
									id="loadpointParamMinCurrent"
									label="Minimum Current"
									example="6A ... 32A"
									class="col-sm-6"
								>
									<PropertyField
										id="loadpointParamMinCurrent"
										v-model="values.minCurrent"
										type="Number"
										unit="A"
										size="w-25 w-min-200"
										class="me-2"
										required
									/>
								</FormRow>
								<FormRow
									id="loadpointParamMaxCurrent"
									label="Maximum Current"
									example="6A ... 32A"
									class="col-sm-6"
								>
									<PropertyField
										id="loadpointParamMaxCurrent"
										v-model="values.maxCurrent"
										type="Number"
										unit="A"
										size="w-25 w-min-200"
										class="me-2"
										required
									/>
								</FormRow>
							</div>

							<div class="row">
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
											{ key: 1, name: '1-phase' },
											{ key: 2, name: '2-phase' },
											{ key: 3, name: '3-phase' },
										]"
										required
									/>
								</FormRow>
								<FormRow
									id="loadpointParamPriority"
									label="Priority"
									help="Higher value means preferred charging with solar surplus."
									class="col-md-6"
								>
									<PropertyField
										id="loadpointParamPriority"
										v-model="values.priority"
										type="Number"
										size="w-75 w-min-200"
										class="me-2"
										:valid-values="priorityOptions"
										required
									/>
								</FormRow>
							</div>
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
							<hr class="mt-3 mb-4" />
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

							<div class="my-4 d-flex justify-content-between">
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
import api from "../../api";

export default {
	name: "LoadpointModal",
	components: { FormRow, PropertyField },
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
			values: {
				title: "",
				phases: 0,
				minCurrent: 0,
				maxCurrent: 0,
				priority: 0,
			},
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
			return [{ key: "", name: "- auto detection -" }, ...this.vehicleOptions];
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
			this.values = {};
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
