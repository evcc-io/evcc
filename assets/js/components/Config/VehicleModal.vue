<template>
	<Teleport to="body">
		<div
			id="vehicleModal"
			ref="modal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
			data-testid="vehicle-modal"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ $t(`config.vehicle.${isNew ? "titleAdd" : "titleEdit"}`) }}
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
							<FormRow id="vehicleTemplate" :label="$t('config.vehicle.template')">
								<select
									id="vehicleTemplate"
									v-model="templateName"
									@change="templateChanged"
									:disabled="!isNew"
									class="form-select w-100"
								>
									<option value="offline">
										{{ $t("config.vehicle.offline") }}
									</option>
									<option disabled>----------</option>
									<optgroup :label="$t('config.vehicle.online')">
										<option
											v-for="option in templateOptions.online"
											:key="option.name"
											:value="option.template"
										>
											{{ option.name }}
										</option>
									</optgroup>
									<optgroup :label="$t('config.vehicle.scooter')">
										<option
											v-for="option in templateOptions.scooter"
											:key="option.name"
											:value="option.template"
										>
											{{ option.name }}
										</option>
									</optgroup>
									<optgroup :label="$t('config.vehicle.generic')">
										<option
											v-for="option in templateOptions.generic"
											:key="option.name"
											:value="option.template"
										>
											{{ option.name }}
										</option>
									</optgroup>
								</select>
							</FormRow>
							<FormRow
								v-for="param in templateParams"
								:id="`vehicleParam${param.Name}`"
								:key="param.Name"
								:optional="!param.Required"
								:label="param.Description || `[${param.Name}]`"
								:help="param.Description === param.Help ? undefined : param.Help"
								:example="param.Example"
							>
								<PropertyField
									:id="`vehicleParam${param.Name}`"
									v-model="values[param.Name]"
									:masked="param.Mask"
									:property="param.Name"
									:type="param.Type"
									class="me-2"
									:required="param.Required"
									:validValues="param.ValidValues"
								/>
							</FormRow>

							<TestResult
								v-if="templateName"
								:success="testSuccess"
								:failed="testFailed"
								:unknown="testUnknown"
								:running="testRunning"
								:result="testResult"
								:error="testError"
								@test="testManually"
							/>

							<div v-if="templateName" class="my-4 d-flex justify-content-between">
								<button
									v-if="isDeletable"
									type="button"
									class="btn btn-link text-danger"
									@click.prevent="remove"
								>
									{{ $t("config.vehicle.delete") }}
								</button>
								<button
									v-else
									type="button"
									class="btn btn-link text-muted"
									data-bs-dismiss="modal"
								>
									{{ $t("config.vehicle.cancel") }}
								</button>
								<button
									type="submit"
									class="btn btn-primary"
									:disabled="testRunning || saving"
									@click.prevent="isNew ? create() : update()"
								>
									<span
										v-if="saving"
										class="spinner-border spinner-border-sm"
										role="status"
										aria-hidden="true"
									></span>
									{{
										testUnknown
											? $t("config.vehicle.validateSave")
											: $t("config.vehicle.save")
									}}
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
import TestResult from "./TestResult.vue";
import api from "../../api";
import test from "./mixins/test";

const initialValues = { type: "template", icon: "car" };

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

export default {
	name: "VehicleModal",
	components: { FormRow, PropertyField, TestResult },
	mixins: [test],
	props: {
		id: Number,
	},
	emits: ["vehicle-changed"],
	data() {
		return {
			isModalVisible: false,
			templates: [],
			products: [],
			saving: false,
			templateName: null,
			template: null,
			values: { ...initialValues },
		};
	},
	computed: {
		templateOptions() {
			return {
				online: this.products.filter((p) => !p.group && p.template !== "offline"),
				generic: this.products.filter((p) => p.group === "generic"),
				scooter: this.products.filter((p) => p.group === "scooter"),
			};
		},
		templateParams() {
			const params = this.template?.Params || [];
			const filteredParams = params.filter((p) => !p.Advanced || p.Name === "icon");
			const adjustedParams = filteredParams.map((p) => {
				if (p.Name === "title" || p.Name === "icon") {
					p.Required = true;
				}
				return p;
			});
			return adjustedParams;
		},
		apiData() {
			return {
				template: this.templateName,
				...this.values,
			};
		},
		isNew() {
			return this.id === undefined;
		},
		isDeletable() {
			return !this.isNew;
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				this.reset();
				this.templateName = "offline";
				this.resetTest();
				this.loadProducts();
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
		},
		templateName() {
			this.loadTemplate();
		},
		values: {
			handler() {
				this.resetTest();
			},
			deep: true,
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
			this.values = { ...initialValues };
			this.resetTest();
		},
		async loadConfiguration() {
			try {
				const vehicle = (await api.get(`config/devices/vehicle/${this.id}`)).data.result;
				this.values = vehicle.config;
				this.applyDefaultsFromTemplate();
				this.templateName = this.values.template;
			} catch (e) {
				console.error(e);
			}
		},
		async loadProducts() {
			try {
				this.products = (await api.get("config/products/vehicle")).data.result;
			} catch (e) {
				console.error(e);
			}
		},
		async loadTemplate() {
			this.template = null;
			try {
				const opts = {
					params: {
						lang: this.$i18n?.locale,
						name: this.templateName,
					},
				};
				this.template = (await api.get("config/templates/vehicle", opts)).data.result;
				this.applyDefaultsFromTemplate();
			} catch (e) {
				console.error(e);
			}
		},
		applyDefaultsFromTemplate() {
			const params = this.template?.Params || [];
			params
				.filter((p) => p.Default && !this.values[p.Name])
				.forEach((p) => {
					this.values[p.Name] = p.Default;
				});
		},
		async create() {
			if (this.testUnknown) {
				const success = await this.test(this.testVehicle);
				if (!success) return;
				await sleep(250);
			}
			this.saving = true;
			try {
				await api.post("config/devices/vehicle", this.apiData);
				this.$emit("vehicle-changed");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("create failed");
			}
			this.saving = false;
		},
		async testManually() {
			await this.test(this.testVehicle);
		},
		async testVehicle() {
			let url = "config/test/vehicle";
			if (!this.isNew) {
				url += `/merge/${this.id}`;
			}
			return await api.post(url, this.apiData);
		},
		async update() {
			if (this.testUnknown) {
				const success = await this.test(this.testVehicle);
				if (!success) return;
				await sleep(250);
			}
			this.saving = true;
			try {
				await api.put(`config/devices/vehicle/${this.id}`, this.apiData);
				this.$emit("vehicle-changed");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await api.delete(`config/devices/vehicle/${this.id}`);
				this.$emit("vehicle-changed");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("delete failed");
			}
		},
		modalVisible() {
			this.isModalVisible = true;
		},
		modalInvisible() {
			this.isModalVisible = false;
		},
		templateChanged() {
			this.reset();
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
</style>
