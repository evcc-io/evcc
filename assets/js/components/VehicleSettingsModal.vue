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
						<h5 class="modal-title">Add New Vehicle 🧪</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div class="container mx-0 px-0">
							<FormRow id="vehicleTemplate" :label="$t('vehicleSettings.template')">
								<select
									id="vehicleTemplate"
									v-model="template"
									class="form-select form-select-sm w-100"
								>
									<option
										v-for="option in vehicleOptions"
										:key="option.name"
										:value="option.value"
									>
										{{ option.name }}
									</option>
								</select>
							</FormRow>
							<FormRow
								v-for="param in templateParams"
								:id="`vehicleParam${param.Name}`"
								:key="param.Name"
								:label="param.Description.EN || `[${param.Name}]`"
							>
								<input
									:id="`vehicleParam${param.Name}`"
									v-model="values[param.Name]"
									:type="param.Mask ? 'password' : 'text'"
									class="w-100 me-2"
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
								<button type="submit" class="btn btn-primary" @click="test">
									{{ $t("vehicleSettings.test") }}
								</button>
							</div>
							<div class="card result">
								<div class="card-body">
									<pre><code>{{ configYaml }}</code></pre>
									<code
										v-if="testResult"
										:class="testSuccess ? 'text-success' : 'text-danger'"
									>
										<hr />
										{{ testResult }}
									</code>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import FormRow from "./FormRow.vue";
import api from "../api";
import YAML from "json-to-pretty-yaml";

export default {
	name: "VehicleSettingsModal",
	components: { FormRow },
	data() {
		return {
			isModalVisible: false,
			templates: [],
			template: null,
			values: {},
			testResult: "",
			testSuccess: false,
		};
	},
	computed: {
		vehicleOptions() {
			const result = [];
			this.templates.forEach((t) => {
				t.Products.forEach((p) => {
					const value = t.Template;
					let name = this.productName(p);
					result.push({ name, value });
				});
			});
			result.sort((a, b) => a.name.localeCompare(b.name));
			return result;
		},
		templateParams() {
			const params = this.templates.find((t) => t.Template === this.template)?.Params || [];
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
				template: this.template,
				...this.values,
			};
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				this.loadTemplates();
			}
		},
		template() {
			this.values = {};
		},
	},
	mounted() {
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hidden.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal?.removeEventListener("hidden.bs.modal", this.modalInvisible);
	},
	methods: {
		productName({ Brand, Description }) {
			const brand = Brand || "";
			const description = Description.Generic || Description.EN || "";
			return `${brand} ${description}`.trim();
		},
		async loadTemplates() {
			try {
				this.templates = (await api.get("config/templates/vehicle")).data.result;
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
