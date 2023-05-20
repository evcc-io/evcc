<template>
	<Teleport to="body">
		<div
			id="vehicleSettingsModal"
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
						<h5 v-if="isNew" class="modal-title">Add New Vehicle</h5>
						<h5 v-else>Edit Vehicle</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div class="container mx-0 px-0">
							<div v-if="isEditable">
								<FormRow
									id="vehicleTemplate"
									:label="$t('vehicleSettings.template')"
								>
									<select
										id="vehicleTemplate"
										v-model="templateName"
										class="form-select w-100"
									>
										<option
											v-for="option in templateOptions"
											:key="option.productName"
											:value="option.template"
										>
											{{ option.productName }}
										</option>
									</select>
								</FormRow>
								<FormRow
									v-for="param in templateParams"
									:id="`vehicleParam${param.Name}`"
									:key="param.Name"
									:optional="!param.Required"
									:label="param.Description || `[${param.Name}]`"
									:small-value="['capacity', 'vin'].includes(param.Name)"
								>
									<InputField
										:id="`vehicleParam${param.Name}`"
										v-model="values[param.Name]"
										:masked="param.Mask"
										:property="param.Name"
										class="me-2"
										:placeholder="param.Example"
										:required="param.Required"
									/>
								</FormRow>
								<div class="buttons d-flex justify-content-between mb-4">
									<button
										type="button"
										class="btn btn-outline-secondary"
										data-bs-dismiss="modal"
									>
										{{ $t("vehicleSettings.cancel") }}
									</button>
									<button
										v-if="!testPerformed"
										type="submit"
										class="btn btn-primary"
										@click="test"
									>
										{{ $t("vehicleSettings.test") }}
									</button>
									<button
										v-else
										type="submit"
										class="btn"
										:class="testSuccess ? 'btn-primary' : 'btn-warning'"
										@click="isNew ? create() : update()"
									>
										{{ isNew ? "Create" : "Update" }}
										<span v-if="!testSuccess"> anyway</span>
									</button>
								</div>
							</div>
							<div v-else>
								<div class="alert alert-warning" role="alert">
									Configuration for
									<code>[type: {{ values.type }}]</code>
									coming later.
								</div>
							</div>
							<div class="card result">
								<div class="card-body evcc-box">
									<pre><code>{{ configYaml }}</code></pre>
									<code
										v-if="testResult"
										:class="testSuccess ? 'text-primary' : 'text-danger'"
									>
										<hr />
										{{ testResult }}
									</code>
								</div>
							</div>
							<div v-if="isDeletable" class="text-center mt-4">
								<button class="btn btn-link text-danger" @click="remove">
									Delete vehicle
								</button>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import FormRow from "../FormRow.vue";
import InputField from "../forms/InputField.vue";
import api from "../../api";
import YAML from "json-to-pretty-yaml";

const initialValues = { type: "template" };

export default {
	name: "VehicleModal",
	components: { FormRow, InputField },
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
			testSuccess: false,
			testPerformed: false,
		};
	},
	computed: {
		templateOptions() {
			const result = Object.entries(this.products).map(([productName, template]) => ({
				productName,
				template,
			}));
			result.sort((a, b) => a.productName.localeCompare(b.productName));
			return result;
		},
		templateParams() {
			const params = this.template?.Params || [];
			return params.filter((p) => !p.Advanced);
		},
		configYaml() {
			return YAML.stringify([
				{
					name: "my_vehicle",
					...this.apiData,
				},
			]);
		},
		apiData() {
			return {
				template: this.templateName,
				...this.values,
			};
		},
		isEditable() {
			return this.values.type === "template";
		},
		isNew() {
			return this.id === undefined;
		},
		isDeletable() {
			return !this.isNew && this.isEditable;
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				this.reset();
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
			this.testPerformed = false;
			this.testSuccess = false;
			this.testResult = "";
		},
		async loadConfiguration() {
			try {
				this.values = (await api.get("config/devices/vehicle")).data.result[this.id - 1];
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
			} catch (e) {
				console.error(e);
			}
		},
		async test() {
			try {
				this.testResult = (await api.post("config/test/vehicle", this.apiData)).data.result;
				this.testSuccess = true;
			} catch (e) {
				console.error(e);
				this.testSuccess = false;
				this.testResult = e.response?.data?.error || e.message;
			}
			this.testPerformed = true;
		},
		async create() {
			try {
				this.result = (await api.post("config/devices/vehicle", this.apiData)).data.result;
				this.$emit("vehicle-changed");
			} catch (e) {
				console.error(e);
				this.testResult = e.response?.data?.error || e.message;
			}
		},
		async update() {
			try {
				this.result = (
					await api.put(`config/devices/vehicle/${this.id}`, this.apiData)
				).data.result;
				this.$emit("vehicle-changed");
			} catch (e) {
				console.error(e);
				this.testResult = e.response?.data?.error || e.message;
			}
		},
		async remove() {
			try {
				this.result = (await api.delete(`config/devices/vehicle/${this.id}`)).data.result;
				this.$emit("vehicle-changed");
			} catch (e) {
				console.error(e);
				this.testResult = e.response?.data?.error || e.message;
				alert("not implemented yet");
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
