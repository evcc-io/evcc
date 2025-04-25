<template>
	<GenericModal
		id="meterModal"
		ref="modal"
		:title="modalTitle"
		data-testid="meter-modal"
		:fade="fade"
		@open="open"
		@close="close"
	>
		<div v-if="!meterType">
			<NewDeviceButton
				v-for="t in typeChoices"
				:key="t"
				:title="$t(`config.meter.option.${t}`)"
				class="mb-4 addButton"
				@click="selectType(t)"
			/>
		</div>
		<form v-else ref="form" class="container mx-0 px-0">
			<p v-if="['aux', 'ext'].includes(meterType)" class="mt-0 mb-4">
				{{ $t(`config.${meterType}.description`) }}
			</p>
			<div v-if="meterType === 'ext'" class="alert alert-warning mb-4" role="alert">
				<strong>Work in Progress:</strong> This feature is not yet available.
			</div>
			<div v-else>
				<FormRow
					v-if="hasDeviceTitle"
					id="meterParamDeviceTitle"
					label="Title"
					help="Will be displayed in the user interface"
				>
					<PropertyField
						id="meterParamDeviceTitle"
						v-model.trim="values.deviceTitle"
						type="String"
						size="w-100"
						class="me-2"
						required
					/>
				</FormRow>
				<FormRow v-if="hasDeviceIcon" id="meterParamDeviceIcon" label="Icon">
					<PropertyField
						id="meterParamDeviceIcon"
						v-model="values.deviceIcon"
						:choice="iconChoices"
						property="icon"
						type="String"
						class="me-2"
						required
					/>
				</FormRow>
				<FormRow id="meterTemplate" :label="$t('config.meter.template')">
					<select
						v-if="isNew"
						id="meterTemplate"
						ref="templateSelect"
						v-model="templateName"
						class="form-select w-100"
						@change="templateChanged"
					>
						<option
							v-for="option in genericOptions"
							:key="option.name"
							:value="option.template"
						>
							{{ option.name }}
						</option>
						<option v-if="genericOptions.length" disabled>──────────</option>
						<option
							v-for="option in templateOptions"
							:key="option.name"
							:value="option.template"
						>
							{{ option.name }}
						</option>
					</select>
					<input
						v-else
						type="text"
						:value="productName"
						disabled
						class="form-control w-100"
					/>
				</FormRow>
				<p v-if="loadingTemplate">{{ $t("config.general.templateLoading") }}</p>
				<Markdown v-if="description" :markdown="description" class="my-4" />
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
				<PropertyEntry
					v-for="param in normalParams"
					:id="`meterParam${param.Name}`"
					:key="param.Name"
					v-bind="param"
					v-model="values[param.Name]"
				/>

				<PropertyCollapsible>
					<template v-if="advancedParams.length" #advanced>
						<PropertyEntry
							v-for="param in advancedParams"
							:id="`meterParam${param.Name}`"
							:key="param.Name"
							v-bind="param"
							v-model="values[param.Name]"
						/>
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
						tabindex="0"
						@click.prevent="remove"
					>
						{{ $t("config.general.delete") }}
					</button>
					<button
						v-else
						type="button"
						class="btn btn-link text-muted"
						data-bs-dismiss="modal"
						tabindex="0"
					>
						{{ $t("config.general.cancel") }}
					</button>
					<button
						type="submit"
						class="btn btn-primary"
						:disabled="testRunning || saving"
						tabindex="0"
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
								? $t("config.general.validateSave")
								: $t("config.general.save")
						}}
					</button>
				</div>
			</div>
		</form>
	</GenericModal>
</template>

<script>
import FormRow from "./FormRow.vue";
import PropertyEntry from "./PropertyEntry.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import TestResult from "./TestResult.vue";
import api from "../../api";
import test from "./mixins/test";
import NewDeviceButton from "./NewDeviceButton.vue";
import Modbus from "./Modbus.vue";
import GenericModal from "../Helper/GenericModal.vue";
import Markdown from "./Markdown.vue";
import PropertyField from "./PropertyField.vue";
const initialValues = { type: "template", deviceTitle: "", deviceIcon: "" };
import { ICONS } from "../VehicleIcon/VehicleIcon.vue";

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

const CUSTOM_FIELDS = ["usage", "modbus"];

const defaultIcons = {
	aux: "smartconsumer",
	ext: "meter",
};

export default {
	name: "MeterModal",
	components: {
		FormRow,
		PropertyEntry,
		PropertyField,
		GenericModal,
		Modbus,
		TestResult,
		NewDeviceButton,
		PropertyCollapsible,
		Markdown,
	},
	mixins: [test],
	props: {
		id: Number,
		name: String,
		type: String,
		typeChoices: {
			type: Array,
			default: () => ["pv", "battery", "aux", "ext"],
		},
		fade: String,
	},
	emits: ["added", "updated", "removed", "close"],
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
			iconChoices: ICONS,
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
		hasDeviceTitle() {
			return ["pv", "battery", "aux", "ext"].includes(this.meterType);
		},
		hasDeviceIcon() {
			return ["aux", "ext"].includes(this.meterType);
		},
		templateOptions() {
			return this.products.filter((p) => p.group !== "generic");
		},
		genericOptions() {
			return this.products.filter((p) => p.group === "generic");
		},
		templateParams() {
			return (this.template?.Params || [])
				.filter((p) => !CUSTOM_FIELDS.includes(p.Name))
				.map((p) => {
					if (this.meterType === "battery" && p.Name === "capacity") {
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
		modbus() {
			const params = this.template?.Params || [];
			return params.find((p) => p.Name === "modbus");
		},
		modbusCapabilities() {
			return this.modbus?.Choice || [];
		},
		modbusDefaults() {
			const { ID, Comset, Baudrate, Port } = this.modbus || {};
			return {
				id: ID,
				comset: Comset,
				baudrate: Baudrate,
				port: Port,
			};
		},
		description() {
			return this.template?.Requirements?.Description;
		},
		productName() {
			return this.values.deviceProduct || this.templateName;
		},
		apiData() {
			return {
				template: this.templateName,
				...this.modbusDefaults,
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
				this.templateName = null;
				this.selectedType = null;
				this.reset();
				this.loadProducts();
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
		},
		meterType(type) {
			this.loadProducts();
			this.values.deviceIcon = defaultIcons[type] || "";
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
	methods: {
		reset(keepTitle = false) {
			const keep = keepTitle
				? { deviceTitle: this.values.deviceTitle, deviceIcon: this.values.deviceIcon }
				: {};
			this.values = { ...initialValues, ...keep };
			this.resetTest();
		},
		async loadConfiguration() {
			try {
				const meter = (await api.get(`config/devices/meter/${this.id}`)).data.result;
				this.values = meter.config;
				// convert structure to flat list
				// TODO: adjust GET response to match POST/PUT formats
				this.values.type = meter.type;
				this.values.deviceTitle = meter.deviceTitle;
				this.values.deviceIcon = meter.deviceIcon;
				this.values.deviceProduct = meter.deviceProduct;
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
						lang: this.$i18n?.locale,
					},
				};
				this.products = (await api.get("config/products/meter", opts)).data.result;
			} catch (e) {
				console.error(e);
			}
		},
		async loadTemplate() {
			this.template = null;
			if (!this.templateName) return;
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
			// persist selected template product
			if (this.template) {
				const select = this.$refs.templateSelect;
				const name = select.options[select.selectedIndex].text;
				this.values.deviceProduct = name;
			}

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
				this.$refs.modal.close();
			} catch (e) {
				this.handleCreateError(e);
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
			return await api.post(url, this.apiData, { timeout: this.testTimeout });
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
				this.$refs.modal.close();
			} catch (e) {
				this.handleUpdateError(e);
			}
			this.saving = false;
		},
		async remove() {
			try {
				await api.delete(`config/devices/meter/${this.id}`);
				this.$emit("removed", this.meterType, this.name);
				this.$refs.modal.close();
			} catch (e) {
				this.handleRemoveError(e);
			}
		},
		open() {
			this.isModalVisible = true;
		},
		close() {
			this.$emit("close");
			this.isModalVisible = false;
		},
		selectType(type) {
			this.selectedType = type;
		},
		templateChanged() {
			this.reset(true);
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
