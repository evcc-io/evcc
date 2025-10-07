<template>
	<GenericModal
		:id="modalId"
		ref="modal"
		:title="modalTitle"
		:data-testid="`${deviceType}-modal`"
		:fade="fade"
		:size="modalSize"
		@open="handleOpen"
		@close="handleClose"
	>
		<form ref="form" class="container mx-0 px-0">
			<slot name="pre-content" :values="values"></slot>

			<template v-if="showMainContent">
				<slot name="description" :values="values"></slot>

				<slot name="before-template" :values="values"></slot>

				<TemplateSelector
					v-if="showTemplateSelector"
					ref="templateSelect"
					v-model="templateName"
					:device-type="deviceType"
					:is-new="isNew"
					:product-name="productName"
					:groups="computedTemplateOptions"
					@change="handleTemplateChange"
				/>

				<YamlEntry
					v-if="showYamlInput"
					v-model="values.yaml"
					:type="deviceType"
					:error-line="test.errorLine"
				/>

				<div v-else>
					<p v-if="loadingTemplate">{{ $t("config.general.templateLoading") }}</p>
					<SponsorTokenRequired v-if="sponsorTokenRequired" />
					<Markdown v-if="description" :markdown="description" class="my-4" />

					<slot name="after-template-info" :values="values"></slot>

					<Modbus
						v-if="modbus"
						v-model:modbus="values['modbus']"
						v-model:id="values['id']"
						v-model:host="values['host']"
						v-model:port="values['port']"
						v-model:device="values['device']"
						v-model:baudrate="values['baudrate']"
						v-model:comset="values['comset']"
						:defaultId="modbus.ID ? Number(modbus.ID) : undefined"
						:defaultComset="modbus.Comset"
						:defaultBaudrate="modbus.Baudrate"
						:defaultPort="modbus.Port"
						:capabilities="modbusCapabilities"
					/>

					<PropertyEntry
						v-for="param in normalParams"
						:id="`${deviceType}Param${param.Name}`"
						:key="param.Name"
						v-bind="param"
						v-model="values[param.Name]"
					/>

					<PropertyCollapsible>
						<template v-if="advancedParams.length" #advanced>
							<PropertyEntry
								v-for="param in advancedParams"
								:id="`${deviceType}Param${param.Name}`"
								:key="param.Name"
								v-bind="param"
								v-model="values[param.Name]"
							/>
						</template>
						<template v-if="$slots['collapsible-more']" #more>
							<slot name="collapsible-more" :values="values"></slot>
						</template>
					</PropertyCollapsible>
				</div>

				<DeviceModalActions
					v-if="showActions"
					:is-deletable="isDeletable"
					:test-state="test"
					:is-saving="saving"
					:is-succeeded="succeeded"
					:is-new="isNew"
					:sponsor-token-required="sponsorTokenRequired"
					@save="handleSave"
					@remove="handleRemove"
					@test="testManually"
				/>
			</template>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import GenericModal, { type ModalFade } from "../../Helper/GenericModal.vue";
import PropertyEntry from "../PropertyEntry.vue";
import PropertyCollapsible from "../PropertyCollapsible.vue";
import Modbus from "./Modbus.vue";
import DeviceModalActions from "./Actions.vue";
import Markdown from "../Markdown.vue";
import SponsorTokenRequired from "./SponsorTokenRequired.vue";
import TemplateSelector, { type TemplateGroup } from "./TemplateSelector.vue";
import YamlEntry from "./YamlEntry.vue";
import { initialTestState, performTest } from "../utils/test";
import sleep from "@/utils/sleep";
import { ConfigType } from "@/types/evcc";
import type { DeviceType } from "@/types/evcc";
import {
	handleError,
	type DeviceValues,
	type Template,
	type TemplateParam,
	type Product,
	type ModbusParam,
	type ModbusCapability,
	type ApiData,
	applyDefaultsFromTemplate,
	createDeviceUtils,
} from "./index";

const CUSTOM_FIELDS = ["modbus"];

export default defineComponent({
	name: "DeviceModalBase",
	components: {
		GenericModal,
		PropertyEntry,
		PropertyCollapsible,
		Modbus,
		DeviceModalActions,
		Markdown,
		SponsorTokenRequired,
		TemplateSelector,
		YamlEntry,
	},
	props: {
		deviceType: { type: String as PropType<DeviceType>, required: true },
		id: Number as PropType<number | undefined>,
		fade: String as PropType<ModalFade>,
		isSponsor: Boolean,
		// Computed/derived props that must be provided by parent
		modalTitle: { type: String, required: true },
		initialValues: { type: Object as PropType<DeviceValues>, required: true },
		customFields: { type: Array as PropType<string[]>, default: () => CUSTOM_FIELDS },
		modalId: { type: String, required: true },
		// Optional: whether to show main content (for multi-step modals like MeterModal)
		showMainContent: { type: Boolean, default: true },
		// Optional: usage parameter for loadProducts (e.g., meter type: "pv", "battery", "aux", "ext")
		usage: String,
		// Optional: custom product name computation
		getProductName: Function as PropType<
			(values: DeviceValues, templateName: string | null) => string
		>,
		// Optional: custom API data transformation
		transformApiData: Function as PropType<(data: ApiData, values: DeviceValues) => ApiData>,
		// Optional: custom template parameter filtering
		filterTemplateParams: Function as PropType<(params: TemplateParam[]) => TemplateParam[]>,
		// Optional: custom defaults application
		applyCustomDefaults: Function as PropType<
			(template: Template | null, values: DeviceValues) => void
		>,
		// Optional: array of field names to preserve when template changes
		preserveOnTemplateChange: Array as PropType<string[]>,
		// Optional: determine if YAML input should be shown
		isYamlInputType: Function as PropType<(type: ConfigType) => boolean>,
		// Optional: provide template options from parent (to avoid circular dependency)
		provideTemplateOptions: Function as PropType<(products: Product[]) => TemplateGroup[]>,
		// Optional: handle template change (receives event and values, allows setting values.yaml)
		onTemplateChange: Function as PropType<(e: Event, values: DeviceValues) => Promise<void>>,
		// Optional: default template to select when opening modal for new devices
		defaultTemplate: String,
	},
	emits: ["added", "updated", "removed", "close", "template-changed"],
	data() {
		return {
			isModalVisible: false,
			products: [] as Product[],
			templateName: null as string | null,
			template: null as Template | null,
			saving: false,
			succeeded: false,
			loadingTemplate: false,
			values: { ...this.initialValues } as DeviceValues,
			test: initialTestState(),
		};
	},
	computed: {
		device() {
			return createDeviceUtils(this.deviceType);
		},
		modalSize(): string | undefined {
			return this.showYamlInput ? "xl" : undefined;
		},
		computedTemplateOptions() {
			if (this.provideTemplateOptions) {
				return this.provideTemplateOptions(this.products);
			}
			return [];
		},
		templateParams() {
			const params = this.template?.Params || [];
			const filtered = params.filter(
				(p) =>
					!this.customFields.includes(p.Name) &&
					(p.Usages ? p.Usages.includes(this.deviceType as any) : true)
			);

			// Allow parent to customize parameter filtering (passes all params for full control)
			if (this.filterTemplateParams) {
				return this.filterTemplateParams(params);
			}

			return filtered;
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
		productName(): string {
			if (this.getProductName) {
				return this.getProductName(this.values, this.templateName);
			}
			return this.values.deviceProduct || this.templateName || "";
		},
		sponsorTokenRequired() {
			const requirements = this.template?.Requirements as any;
			return requirements?.EVCC?.includes("sponsorship") && !this.isSponsor;
		},
		apiData(): ApiData {
			let data: ApiData = {
				...this.modbusDefaults,
				...this.values,
			};
			if (this.values.type === ConfigType.Template && this.templateName) {
				data["template"] = this.templateName;
			} else {
				// Remove template field if not using template type or if no template selected
				delete data["template"];
			}
			if (this.showYamlInput) {
				// Icon is extracted from yaml on GET for UI purpose only. Don't write it back.
				delete data["icon"];
			}

			// Allow parent to transform API data
			if (this.transformApiData) {
				data = this.transformApiData(data, this.values);
			}

			return data;
		},
		isNew() {
			return this.id === undefined;
		},
		isDeletable() {
			return !this.isNew;
		},
		showActions() {
			return this.templateName || this.showYamlInput;
		},
		showYamlInput() {
			if (this.isYamlInputType) {
				return this.isYamlInputType(this.values.type);
			}
			return this.values.type === ConfigType.Custom;
		},
		showTemplateSelector() {
			return this.computedTemplateOptions.length > 0;
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				this.templateName =
					this.isNew && this.defaultTemplate ? this.defaultTemplate : null;
				this.reset();
				this.test = initialTestState();
				this.succeeded = false;
				this.loadProducts();
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
		},
		templateName(newValue, oldValue) {
			// Reset values when template changes (except on initial load or when switching to YAML input)
			// YAML input types set values.type and values.yaml in handleTemplateChange callback
			const isYamlInput =
				this.isYamlInputType && newValue && this.isYamlInputType(newValue as any);
			if (oldValue != null && !isYamlInput) {
				if (this.preserveOnTemplateChange) {
					const preserved: Record<string, any> = {};
					this.preserveOnTemplateChange.forEach((field) => {
						preserved[field] = this.values[field];
					});
					this.reset();
					this.preserveOnTemplateChange.forEach((field) => {
						this.values[field] = preserved[field];
					});
				} else {
					this.reset();
				}
			}
			this.loadTemplate();
		},
		usage() {
			// Reload products when usage changes (e.g., meter type selection)
			this.loadProducts();
			// Apply defaults when usage changes (e.g., set default icon for meter type)
			this.applyDefaults();
		},
		values: {
			handler() {
				this.test = initialTestState();
			},
			deep: true,
		},
	},
	methods: {
		reset() {
			this.values = { ...this.initialValues } as DeviceValues;
			this.test = initialTestState();
		},
		async loadConfiguration() {
			try {
				const device = await this.device.load(this.id!);
				this.values = device.config;
				// convert structure to flat list
				// TODO: adjust GET response to match POST/PUT formats
				this.values.type = device.type;
				this.values.deviceProduct = device.deviceProduct;
				if (device.deviceTitle !== undefined) {
					this.values["deviceTitle"] = device.deviceTitle;
				}
				if (device.deviceIcon !== undefined) {
					this.values["deviceIcon"] = device.deviceIcon;
				}
				this.applyDefaults();
				this.templateName = this.values.template;
			} catch (e) {
				console.error(e);
			}
		},
		applyDefaults() {
			applyDefaultsFromTemplate(this.template, this.values);

			// Allow parent to apply custom defaults
			if (this.applyCustomDefaults) {
				this.applyCustomDefaults(this.template, this.values);
			}
		},
		async loadProducts() {
			if (!this.isModalVisible) {
				return;
			}
			try {
				this.products = await this.device.loadProducts(this.$i18n?.locale, this.usage);
			} catch (e) {
				console.error(e);
			}
		},
		async loadTemplate() {
			this.template = null;
			if (!this.templateName || this.showYamlInput) return;
			this.loadingTemplate = true;
			try {
				this.template = await this.device.loadTemplate(
					this.templateName,
					this.$i18n?.locale
				);
				this.applyDefaults();
			} catch (e) {
				console.error(e);
			}
			this.loadingTemplate = false;
		},
		async create(force = false) {
			if (this.test.isUnknown && !force) {
				const success = await performTest(
					this.test,
					this.testDevice,
					this.$refs["form"] as HTMLFormElement
				);
				if (!success) {
					return;
				}
			}

			// persist selected template product
			if (this.template && this.$refs["templateSelect"]) {
				this.values.deviceProduct = (this.$refs["templateSelect"] as any).getProductName();
			}

			this.saving = true;
			try {
				const { name } = await this.device.create(this.apiData, force);
				this.saving = false;
				this.succeeded = true;
				await sleep(1000);
				this.$emit("added", name);
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "create failed");
				this.saving = false;
			}
		},
		async testManually() {
			await performTest(this.test, this.testDevice, this.$refs["form"] as HTMLFormElement);
		},
		async testDevice() {
			return this.device.test(this.id, this.apiData);
		},
		async update(force = false) {
			console.log("update called", { force, isUnknown: this.test.isUnknown, id: this.id });
			if (this.test.isUnknown && !force) {
				const success = await performTest(
					this.test,
					this.testDevice,
					this.$refs["form"] as HTMLFormElement
				);
				console.log("test result", success);
				if (!success) {
					return;
				}
			}
			this.saving = true;
			try {
				console.log("calling device.update", this.apiData);
				await this.device.update(this.id!, this.apiData, force);
				console.log("update succeeded, closing modal");
				this.saving = false;
				this.succeeded = true;
				await sleep(1000);
				this.$emit("updated");
				(this.$refs["modal"] as any).close();
			} catch (e) {
				console.error("update failed", e);
				handleError(e, "update failed");
				this.saving = false;
			}
		},
		async remove() {
			try {
				await this.device.remove(this.id!);
				this.$emit("removed");
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "remove failed");
			}
		},
		handleOpen() {
			this.isModalVisible = true;
		},
		handleClose() {
			this.$emit("close");
			this.isModalVisible = false;
		},
		async handleTemplateChange(e: Event) {
			// Allow parent to handle custom logic (e.g., loading default YAML)
			if (this.onTemplateChange) {
				await this.onTemplateChange(e, this.values);
			}
			// Emit to parent for notification
			this.$emit("template-changed", e);
		},
		handleSave(force: boolean) {
			if (this.isNew) {
				this.create(force);
			} else {
				this.update(force);
			}
		},
		handleRemove() {
			this.remove();
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
</style>
