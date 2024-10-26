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
							<p v-if="loadingTemplate">Loading ...</p>
							<Markdown v-if="description" :markdown="description" class="my-4" />
							<PropertyEntry
								v-for="param in normalParams"
								:key="param.Name"
								:id="`vehicleParam${param.Name}`"
								v-bind="param"
								v-model="values[param.Name]"
							/>

							<PropertyCollapsible>
								<template v-if="advancedParams.length" #advanced>
									<PropertyEntry
										v-for="param in advancedParams"
										:key="param.Name"
										:id="`vehicleParam${param.Name}`"
										v-bind="param"
										v-model="values[param.Name]"
									/>
								</template>
								<template #more>
									<h6 class="mt-3">Charging settings</h6>
									<FormRow
										id="vehicleParamMode"
										label="Default mode"
										help="Charging point mode when connecting this vehicle."
									>
										<PropertyField
											id="vehicleParamMode"
											v-model="values.mode"
											type="String"
											class="w-100"
											:valid-values="[
												{ key: 'off', name: $t('main.mode.off') },
												{ key: 'pv', name: $t('main.mode.pv') },
												{ key: 'minpv', name: $t('main.mode.minpv') },
												{ key: 'now', name: $t('main.mode.now') },
											]"
										/>
									</FormRow>
									<FormRow
										id="vehicleParamPhases"
										label="Maximum phases"
										help="How many phases can this vehicle charge with? Used to calculate required minimum solar surplus and plan duration."
									>
										<SelectGroup
											id="vehicleParamPhases"
											class="w-100"
											v-model="values.phases"
											:options="[
												{ name: '1-phase', value: '1' },
												{ name: '2-phases', value: '2' },
												{ name: '3-phases', value: undefined },
											]"
											transparent
											equal-width
										/>
									</FormRow>
									<div class="row mb-3">
										<FormRow
											id="vehicleParamMinCurrent"
											label="Minimum current"
											class="col-sm-6 mb-sm-0"
											:help="
												values.minCurrent && values.minCurrent < 6
													? 'Only go below 6 A if you know what you\'re doing.'
													: null
											"
										>
											<PropertyField
												id="vehicleParamMinCurrent"
												v-model="values.minCurrent"
												type="Float"
												unit="A"
												size="w-25 w-min-200"
												class="me-2"
											/>
										</FormRow>

										<FormRow
											id="vehicleParamMaxCurrent"
											label="Maximum current"
											class="col-sm-6 mb-sm-0"
											:help="
												values.minCurrent &&
												values.maxCurrent &&
												values.maxCurrent < values.minCurrent
													? 'Must be greater than minimum current.'
													: null
											"
										>
											<PropertyField
												id="vehicleParamMaxCurrent"
												v-model="values.maxCurrent"
												type="Float"
												unit="A"
												size="w-25 w-min-200"
												class="me-2"
											/>
										</FormRow>
									</div>

									<!-- todo: only show when multiple loadpoints exist -->
									<FormRow
										id="vehicleParamPriority"
										label="Priority"
										help="Changes the charging point priority when connecting this vehicle."
									>
										<PropertyField
											id="vehicleParamPriority"
											v-model="values.priority"
											type="Number"
											size="w-100"
											class="me-2"
											:valid-values="priorityOptions"
											required
										/>
									</FormRow>

									<FormRow
										id="vehicleParamIdentifiers"
										label="RFID identifiers"
										help="List of RFID strings to identify the vehicle. One per line. See the current identifier on the configuration overview."
									>
										<PropertyField
											id="vehicleParamIdentifiers"
											v-model="values.identifiers"
											type="StringList"
											property="identifiers"
											size="w-100"
											class="me-2"
										/>
									</FormRow>
								</template>
							</PropertyCollapsible>

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
import SelectGroup from "../SelectGroup.vue";
import PropertyEntry from "./PropertyEntry.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import Markdown from "./Markdown.vue";
import api from "../../api";
import test from "./mixins/test";

const initialValues = { type: "template", icon: "car" };

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

const CUSTOM_FIELDS = ["minCurrent", "maxCurrent", "priority", "identifiers", "phases", "mode"];

export default {
	name: "VehicleModal",
	components: {
		FormRow,
		PropertyField,
		TestResult,
		SelectGroup,
		PropertyCollapsible,
		PropertyEntry,
		Markdown,
	},
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
			loadingTemplate: false,
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
			return (this.template?.Params || [])
				.filter((p) => !CUSTOM_FIELDS.includes(p.Name))
				.map((p) => {
					if (p.Name === "title" || p.Name === "icon") {
						p.Required = true;
						p.Advanced = false;
					}
					return p;
				});
		},
		normalParams() {
			return this.templateParams.filter((p) => !p.Advanced);
		},
		advancedParams() {
			return this.templateParams.filter((p) => p.Advanced);
		},
		description() {
			return this.template?.Requirements?.Description;
		},
		apiData() {
			const data = {
				template: this.templateName,
				...this.values,
			};
			// trim and remove empty lines
			if (Array.isArray(data.identifiers)) {
				data.identifiers = data.identifiers.map((i) => i.trim()).filter((i) => i);
			}
			return data;
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
			result[0].key = undefined;
			result[10].name = "10 (highest)";
			return result;
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
			this.loadingTemplate = true;
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
			this.loadingTemplate = false;
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
import type MarkdownVue from "./Markdown.vue";
