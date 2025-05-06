<template>
	<GenericModal
		id="meterModal"
		ref="modal"
		:title="modalTitle"
		data-testid="meter-modal"
		:fade="fade"
		:size="modalSize"
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
			<p v-if="hasDescription" class="mt-0 mb-4">
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

				<TemplateSelector
					v-if="isNew"
					ref="templateSelect"
					v-model="templateName"
					device-type="meter"
					:is-new="isNew"
					:product-name="productName"
					:groups="templateOptions"
					@change="templateChanged"
				/>

				<YamlEntry v-if="values.type === 'custom'" v-model="values.yaml" type="meter" />
				<div v-else>
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
						:defaultId="modbus.ID ? Number(modbus.ID) : undefined"
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
				</div>

				<DeviceModalActions
					v-if="showActions"
					:is-deletable="isDeletable"
					:test-state="test"
					:is-saving="saving"
					@save="isNew ? create() : update()"
					@remove="remove"
					@test="testManually"
				/>
			</div>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import FormRow from "./FormRow.vue";
import PropertyEntry from "./PropertyEntry.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import api from "@/api";
import NewDeviceButton from "./NewDeviceButton.vue";
import Modbus from "./DeviceModal/Modbus.vue";
import DeviceModalActions from "./DeviceModal/Actions.vue";
import GenericModal from "../Helper/GenericModal.vue";
import Markdown from "./Markdown.vue";
import PropertyField from "./PropertyField.vue";
import TemplateSelector from "./DeviceModal/TemplateSelector.vue";
import YamlEntry from "./DeviceModal/YamlEntry.vue";
import { ICONS } from "../VehicleIcon/VehicleIcon.vue";
import { initialTestState, performTest } from "./utils/test";
import {
	handleError,
	timeout,
	ConfigType,
	type DeviceValues,
	type Template,
	type Product,
	type ModbusParam,
	type ModbusCapability,
} from "./DeviceModal";
import defaultYaml from "./defaultYaml/meter.yaml?raw";

const initialValues = { type: ConfigType.Template, deviceTitle: "", deviceIcon: "" };

function sleep(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

const CUSTOM_FIELDS = ["usage", "modbus"];

const defaultIcons: Record<string, string> = {
	aux: "smartconsumer",
	ext: "meter",
};

type MeterDeviceValues = DeviceValues & {
	deviceTitle: string;
	deviceIcon: string;
	modbus?: any;
	id?: any;
	host?: any;
	port?: any;
	device?: any;
	baudrate?: any;
	comset?: any;
	yaml?: string;
};

export default defineComponent({
	name: "MeterModal",
	components: {
		FormRow,
		PropertyEntry,
		PropertyField,
		GenericModal,
		Modbus,
		NewDeviceButton,
		PropertyCollapsible,
		Markdown,
		TemplateSelector,
		YamlEntry,
		DeviceModalActions,
	},
	props: {
		id: Number,
		name: String,
		type: {
			type: String as () => string | undefined,
			default: undefined,
		},
		typeChoices: {
			type: Array as () => string[],
			default: () => ["pv", "battery", "aux", "ext"],
		},
		fade: String,
	},
	emits: ["added", "updated", "removed", "close"],
	data() {
		return {
			isModalVisible: false,
			templates: [] as Template[],
			products: [] as Product[],
			templateName: null as string | null,
			template: null as Template | null,
			saving: false,
			selectedType: null as string | null,
			loadingTemplate: false,
			iconChoices: ICONS,
			values: { ...initialValues } as MeterDeviceValues,
			test: initialTestState(),
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
			return ["pv", "battery", "aux", "ext"].includes(this.meterType || "");
		},
		hasDeviceIcon() {
			return ["aux", "ext"].includes(this.meterType || "");
		},
		templateOptions() {
			return [
				{
					label: "generic",
					options: this.products.filter((p) => p.group === "generic"),
				},
				{
					label: "specific",
					options: this.products.filter((p) => p.group !== "generic"),
				},
			];
		},
		templateParams() {
			const params = (this.template?.Params || [])
				.filter((p) => !CUSTOM_FIELDS.includes(p.Name))
				.map((p) => {
					if (this.meterType === "battery" && p.Name === "capacity") {
						p.Advanced = false;
					}
					return p;
				});
			return params;
		},
		normalParams() {
			return this.templateParams.filter((p) => !p.Advanced && !p.Deprecated);
		},
		advancedParams() {
			return this.templateParams.filter((p) => p.Advanced || p.Deprecated);
		},
		modbus(): ModbusParam | undefined {
			const params = this.template?.Params || [];
			return (params as ModbusParam[]).find((p) => p.Name === "modbus");
		},
		modbusCapabilities() {
			return (this.modbus?.Choice || []) as ModbusCapability[];
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
			return this.values.deviceProduct || this.templateName || "";
		},
		apiData() {
			const data: Record<string, any> = {
				...this.modbusDefaults,
				...this.values,
			};
			if (this.values.type === ConfigType.Template) {
				data["template"] = this.templateName;
				data["usage"] = this.meterType || undefined;
			}
			return data;
		},
		isNew() {
			return this.id === undefined;
		},
		isDeletable() {
			return !this.isNew;
		},
		modalSize() {
			return this.values.type === ConfigType.Custom ? "xl" : undefined;
		},
		showActions() {
			return this.templateName || this.values.type === ConfigType.Custom;
		},
		hasDescription() {
			return ["ext", "aux"].includes(this.meterType || "");
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				this.templateName = null;
				this.selectedType = null;
				this.reset();
				this.test = initialTestState();
				this.loadProducts();
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
		},
		meterType(type) {
			if (!type) return;
			this.loadProducts();
			this.values.deviceIcon = defaultIcons[type] || "";
		},
		templateName() {
			this.loadTemplate();
		},
		values: {
			handler() {
				this.test = initialTestState();
			},
			deep: true,
		},
	},
	methods: {
		reset(keepTitle = false) {
			const keep = keepTitle
				? { deviceTitle: this.values.deviceTitle, deviceIcon: this.values.deviceIcon }
				: {};
			this.values = { ...initialValues, ...keep } as MeterDeviceValues;
			this.test = initialTestState();
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
			if (this.template && this.$refs["templateSelect"]) {
				this.values.deviceProduct = (this.$refs["templateSelect"] as any).getProductName();
			}

			if (this.test.isUnknown) {
				const success = await performTest(
					this.test,
					this.testMeter,
					this.$refs["form"] as HTMLFormElement
				);
				if (!success) return;
				await sleep(100);
			}
			this.saving = true;
			try {
				const response = await api.post("config/devices/meter", this.apiData);
				const { name } = response.data.result;
				this.$emit("added", this.meterType, name);
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "create failed");
			}
			this.saving = false;
		},
		async testManually() {
			await performTest(this.test, this.testMeter, this.$refs["form"] as HTMLFormElement);
		},
		async testMeter() {
			let url = "config/test/meter";
			if (!this.isNew) {
				url += `/merge/${this.id}`;
			}
			return await api.post(url, this.apiData, { timeout });
		},
		async update() {
			if (this.test.isUnknown) {
				const success = await performTest(
					this.test,
					this.testMeter,
					this.$refs["form"] as HTMLFormElement
				);
				if (!success) return;
				await sleep(250);
			}
			this.saving = true;
			try {
				await api.put(`config/devices/meter/${this.id}`, this.apiData);
				this.$emit("updated");
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await api.delete(`config/devices/meter/${this.id}`);
				this.$emit("removed", this.meterType, this.name);
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "remove failed");
			}
		},
		open() {
			this.isModalVisible = true;
		},
		close() {
			this.$emit("close");
			this.isModalVisible = false;
		},
		selectType(type: string) {
			this.selectedType = type;
		},
		templateChanged() {
			this.reset(true);
			if (this.templateName === ConfigType.Custom) {
				this.values.type = ConfigType.Custom;
				this.values.yaml = defaultYaml;
			}
		},
	},
});
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
