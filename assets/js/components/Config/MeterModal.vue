<template>
	<Teleport to="body">
		<div
			id="meterModal"
			ref="modal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
			data-testid="meter-modal"
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
						<div v-if="!meterType">
							<AddDeviceButton
								title="Add solar meter"
								class="mb-4 addButton"
								@click="selectType('pv')"
							/>
							<AddDeviceButton
								title="Add battery meter"
								class="addButton"
								@click="selectType('battery')"
							/>
						</div>
						<form v-else ref="form" class="container mx-0 px-0">
							<FormRow id="meterTemplate" :label="$t('config.meter.template')">
								<select
									id="meterTemplate"
									v-model="templateName"
									@change="templateChanged"
									:disabled="!isNew"
									class="form-select w-100"
								>
									<option
										v-for="option in genericOptions"
										:key="option.name"
										:value="option.template"
									>
										{{ option.name }}
									</option>
									<option v-if="genericOptions.length" disabled>
										──────────
									</option>
									<option
										v-for="option in templateOptions"
										:key="option.name"
										:value="option.template"
									>
										{{ option.name }}
									</option>
								</select>
							</FormRow>
							<p v-if="loadingTemplate">Loading ...</p>
							<Modbus
								v-if="modbus"
								v-model:modbus="values.modbus"
								v-model:id="values.id"
								v-model:host="values.host"
								v-model:port="values.port"
								v-model:device="values.device"
								v-model:baudrate="values.baudrate"
								v-model:comset="values.comset"
								:defaultId="modbus.ID"
								:defaultComset="modbus.Comset"
								:defaultBaudrate="modbus.Baudrate"
								:defaultPort="modbus.Port"
								:capabilities="modbusCapabilities"
							/>
							<FormRow
								v-for="param in templateParams"
								:id="`meterParam${param.Name}`"
								:key="param.Name"
								:optional="!param.Required"
								:label="param.Description || `[${param.Name}]`"
								:help="param.Description === param.Help ? undefined : param.Help"
								:example="param.Example"
							>
								<PropertyField
									:id="`meterParam${param.Name}`"
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
									{{ $t("config.meter.delete") }}
								</button>
								<button
									v-else
									type="button"
									class="btn btn-link text-muted"
									data-bs-dismiss="modal"
								>
									{{ $t("config.meter.cancel") }}
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
											? $t("config.meter.validateSave")
											: $t("config.meter.save")
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
import AddDeviceButton from "./AddDeviceButton.vue";
import Modbus from "./Modbus.vue";

const initialValues = { type: "template" };

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

export default {
	name: "MeterModal",
	components: { FormRow, PropertyField, Modbus, TestResult, AddDeviceButton },
	mixins: [test],
	props: {
		id: Number,
		name: String,
		type: String,
	},
	emits: ["added", "updated", "removed"],
	data() {
		return {
			isModalVisible: false,
			templates: [],
			products: [],
			templateName: null,
			template: null,
			saving: false,
			selectedType: null,
			loadingTemplate: false,
			values: { ...initialValues },
		};
	},
	computed: {
		modalTitle() {
			if (this.isNew) {
				if (this.meterType) {
					return this.$t(`config.${this.meterType}.titleAdd`);
				} else {
					return this.$t("config.meter.titleChoice");
				}
			}
			return this.$t(`config.${this.meterType}.titleEdit`);
		},
		meterType() {
			return this.type || this.selectedType;
		},
		templateOptions() {
			return this.products.filter((p) => p.group !== "generic");
		},
		genericOptions() {
			return this.products.filter((p) => p.group === "generic");
		},
		templateParams() {
			const params = this.template?.Params || [];
			return (
				params
					// deprecated fields
					.filter((p) => !p.Deprecated)
					// remove usage option
					.filter((p) => p.Name !== "usage")
					// remove modbus, handles separately
					.filter((p) => p.Name !== "modbus")
					// capacity only for battery meters
					.filter((p) => this.meterType === "battery" || p.Name !== "capacity")
			);
		},
		modbus() {
			const params = this.template?.Params || [];
			return params.find((p) => p.Name === "modbus");
		},
		modbusCapabilities() {
			return this.modbus?.Choice || [];
		},
		apiData() {
			return {
				template: this.templateName,
				...this.values,
				usage: this.meterType,
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
				this.selectedType = null;
				this.reset();
				this.resetTest();
				this.loadProducts();
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
		},
		meterType() {
			this.loadProducts();
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
				const meter = (await api.get(`config/devices/meter/${this.id}`)).data.result;
				this.values = meter.config;
				this.applyDefaultsFromTemplate();
				this.templateName = this.values.template;
			} catch (e) {
				console.error(e);
			}
		},
		async loadProducts() {
			if (!this.isModalVisible || !this.meterType) {
				return;
			}
			try {
				const opts = {
					params: {
						usage: this.meterType,
					},
				};
				this.products = (await api.get("config/products/meter", opts)).data.result;
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
				const result = await api.get("config/templates/meter", opts);
				this.template = result.data.result;
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
				const success = await this.test(this.testMeter);
				if (!success) return;
				await sleep(100);
			}
			this.saving = true;
			try {
				const response = await api.post("config/devices/meter", this.apiData);
				const { name } = response.data.result;
				this.$emit("added", this.meterType, name);
				this.$emit("updated");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("create failed");
			}
			this.saving = false;
		},
		async testManually() {
			await this.test(this.testMeter);
		},
		async testMeter() {
			let url = "config/test/meter";
			if (!this.isNew) {
				url += `/merge/${this.id}`;
			}
			return await api.post(url, this.apiData);
		},
		async update() {
			if (this.testUnknown) {
				const success = await this.test(this.testMeter);
				if (!success) return;
				await sleep(250);
			}
			this.saving = true;
			try {
				await api.put(`config/devices/meter/${this.id}`, this.apiData);
				this.$emit("updated");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await api.delete(`config/devices/meter/${this.id}`);
				this.$emit("removed", this.meterType, this.name);
				this.$emit("updated");
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
		selectType(type) {
			this.selectedType = type;
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
.addButton {
	min-height: auto;
}
</style>
