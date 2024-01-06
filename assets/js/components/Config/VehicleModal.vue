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
									:disabled="!isNew"
									class="form-select w-100"
									@change="templateChanged"
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
								:small-value="['capacity'].includes(param.Name)"
								:example="param.Example"
							>
								<PropertyField
									:id="`vehicleParam${param.Name}`"
									v-model="values[param.Name]"
									:masked="param.Mask"
									:property="param.Name"
									class="me-2"
									:required="param.Required"
									:validValues="param.ValidValues"
								/>
							</FormRow>

							<div
								v-if="templateName"
								class="alert my-4"
								:class="{
									'alert-secondary': testUnknown || testRunning,
									'alert-success': testSuccess,
									'alert-danger': testFailed,
								}"
								role="alert"
							>
								<div class="d-flex justify-content-between align-items-center">
									<div>
										{{ $t("config.validation.label") }}:
										<span v-if="testUnknown">{{
											$t("config.validation.unknown")
										}}</span>
										<span v-if="testRunning">{{
											$t("config.validation.running")
										}}</span>
										<strong v-if="testSuccess">{{
											$t("config.validation.success")
										}}</strong>
										<strong v-if="testFailed">{{
											$t("config.validation.failed")
										}}</strong>
									</div>
									<a href="#" class="alert-link" @click.prevent="test">
										{{ $t("config.validation.validate") }}
									</a>
								</div>
								<hr v-if="testResult" />
								<div v-if="testResult">
									{{ testResult }}
								</div>
							</div>

							<div v-if="templateName" class="my-4">
								<button
									type="submit"
									class="btn btn-primary me-3"
									:disabled="testRunning"
									@click.prevent="isNew ? create() : update()"
								>
									{{
										testUnknown
											? $t("config.vehicle.validateSave")
											: $t("config.vehicle.save")
									}}
								</button>
								<button
									type="button"
									class="btn btn-link text-muted"
									data-bs-dismiss="modal"
								>
									{{ $t("config.vehicle.cancel") }}
								</button>
							</div>

							<div v-if="isDeletable" class="text-center mt-4">
								<button
									type="button"
									class="btn btn-link text-danger"
									@click.prevent="remove"
								>
									{{ $t("config.vehicle.delete") }}
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

const initialValues = { type: "template", icon: "car" };

const TEST_UNKNOWN = "unknown";
const TEST_SUCCESS = "success";
const TEST_FAILED = "failed";
const TEST_RUNNING = "running";

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

export default {
	name: "VehicleModal",
	components: { FormRow, PropertyField },
	props: {
		id: Number,
	},
	emits: ["vehicle-changed"],
	data() {
		return {
			isModalVisible: false,
			templates: [],
			products: [],
			templateName: null,
			template: null,
			values: { ...initialValues },
			testResult: "",
			testState: TEST_UNKNOWN,
		};
	},
	computed: {
		testRunning() {
			return this.testState === TEST_RUNNING;
		},
		testSuccess() {
			return this.testState === TEST_SUCCESS;
		},
		testFailed() {
			return this.testState === TEST_FAILED;
		},
		testUnknown() {
			return this.testState === TEST_UNKNOWN;
		},
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
		resetTest() {
			this.testState = TEST_UNKNOWN;
			this.testResult = null;
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
			try {
				const opts = {
					params: {
						lang: this.$i18n.locale,
						name: this.templateName,
					},
				};
				this.template = (await api.get("config/templates/vehicle", opts)).data.result;
				this.applyDefaultsFromTemplate();
			} catch (e) {
				console.error(e);
			}
		},
		templateChanged() {
			this.reset();
		},
		applyDefaultsFromTemplate() {
			const params = this.template?.Params || [];
			params
				.filter((p) => p.Default && !this.values[p.Name])
				.forEach((p) => {
					this.values[p.Name] = p.Default;
				});
		},
		async test() {
			if (!this.$refs.form.reportValidity()) return false;
			this.testState = TEST_RUNNING;
			try {
				let url = "config/test/vehicle";
				if (!this.isNew) {
					url += `/${this.id}`;
				}
				await api.post(url, this.apiData);
				this.testState = TEST_SUCCESS;
				this.testResult = null;
				return true;
			} catch (e) {
				console.error(e);
				this.testState = TEST_FAILED;
				this.testResult = e.response?.data?.error || e.message;
			}
			return false;
		},
		async create() {
			if (this.testUnknown) {
				const success = await this.test();
				if (!success) return;
				await sleep(250);
			}
			try {
				await api.post("config/devices/vehicle", this.apiData);
				this.$emit("vehicle-changed");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("create failed");
			}
		},
		async update() {
			if (this.testUnknown) {
				const success = await this.test();
				if (!success) return;
				await sleep(250);
			}
			try {
				await api.put(`config/devices/vehicle/${this.id}`, this.apiData);
				this.$emit("vehicle-changed");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("update failed");
			}
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
