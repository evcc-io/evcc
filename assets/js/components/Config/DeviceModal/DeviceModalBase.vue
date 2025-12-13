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
		@visibilitychange="handleVisibilityChange"
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

				<p v-if="showDeprecatedWarning" class="text-danger">
					{{ $t("config.general.typeDeprecated", { type: values.type }) }}
				</p>

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

					<div v-if="authRequired">
						<PropertyEntry
							v-for="param in authParams"
							:id="`${deviceType}Param${param.Name}`"
							:key="param.Name"
							v-bind="param"
							v-model="values[param.Name]"
							:service-values="serviceValues[param.Name]"
						/>

						<div v-if="auth.code">
							<hr class="my-5" />
							<AuthCodeDisplay
								:id="`${deviceType}AuthCode`"
								:code="auth.code"
								:expiry="auth.expiry"
							/>
						</div>

						<p v-if="auth.error" class="text-danger">{{ auth.error }}</p>

						<div
							class="my-4 d-flex align-items-stretch justify-content-sm-between align-items-sm-baseline flex-column-reverse flex-sm-row gap-2"
						>
							<!-- delete / cancel -->
							<button
								v-if="isDeletable"
								type="button"
								class="btn btn-link text-danger align-self-start"
								tabindex="0"
								@click.prevent="handleRemove"
							>
								{{ $t("config.general.delete") }}
							</button>
							<button
								v-else
								type="button"
								class="btn btn-link text-muted align-self-start"
								data-bs-dismiss="modal"
								tabindex="0"
							>
								{{ $t("config.general.cancel") }}
							</button>
							<!-- perform auth -->
							<AuthConnectButton
								:provider-url="auth.providerUrl ?? undefined"
								:loading="auth.loading"
								@prepare="checkAuthStatus"
							/>
						</div>
					</div>
					<div v-else>
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
							component-id="device"
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
							:service-values="serviceValues[param.Name]"
						/>

						<PropertyCollapsible>
							<template v-if="advancedParams.length" #advanced>
								<PropertyEntry
									v-for="param in advancedParams"
									:id="`${deviceType}Param${param.Name}`"
									:key="param.Name"
									v-bind="param"
									v-model="values[param.Name]"
									:service-values="serviceValues[param.Name]"
								/>
							</template>
							<template v-if="$slots['collapsible-more']" #more>
								<slot name="collapsible-more" :values="values"></slot>
							</template>
						</PropertyCollapsible>
					</div>
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
import AuthCodeDisplay from "../AuthCodeDisplay.vue";
import AuthConnectButton from "../AuthConnectButton.vue";
import { initialTestState, performTest } from "../utils/test";
import { initialAuthState, prepareAuthLogin } from "../utils/authProvider";
import sleep from "@/utils/sleep";
import { ConfigType } from "@/types/evcc";
import type { DeviceType, Timeout } from "@/types/evcc";
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
	fetchServiceValues,
} from "./index";
import deepEqual from "@/utils/deepEqual";

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
		AuthCodeDisplay,
		AuthConnectButton,
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
		// Optional: determine if a config type is deprecated
		isTypeDeprecated: Function as PropType<(type: ConfigType) => boolean>,
		// Optional: provide template options from parent (to avoid circular dependency)
		provideTemplateOptions: Function as PropType<(products: Product[]) => TemplateGroup[]>,
		// Optional: handle template change (receives event and values, allows setting values.yaml)
		onTemplateChange: Function as PropType<(e: Event, values: DeviceValues) => void>,
		// Optional: default template to select when opening modal for new devices
		defaultTemplate: String,
		// Optional: callback after configuration is loaded (receives values)
		onConfigurationLoaded: Function as PropType<(values: DeviceValues) => void>,
		// Optional: external template selection control (for parent to reset template)
		externalTemplate: String as PropType<string | null>,
	},
	emits: ["added", "updated", "removed", "close", "template-changed", "update:externalTemplate"],
	data() {
		return {
			isModalVisible: false,
			products: [] as Product[],
			templateName: null as string | null,
			template: null as Template | null,
			saving: false,
			auth: initialAuthState(),
			succeeded: false,
			loadingTemplate: false,
			values: { ...this.initialValues } as DeviceValues,
			test: initialTestState(),
			serviceValues: {} as Record<string, string[]>,
			serviceValuesTimer: null as Timeout | null,
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
		authParams() {
			const { params = [] } = this.template?.Auth ?? {};
			return this.templateParams.filter((p) => params.includes(p.Name));
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
			return (this.templateName && !this.authRequired) || this.showYamlInput;
		},
		showYamlInput() {
			return this.isYamlInputTypeByValue(this.values.type);
		},
		showTemplateSelector() {
			return this.computedTemplateOptions.length > 0;
		},
		showDeprecatedWarning() {
			return this.isTypeDeprecated && this.isTypeDeprecated(this.values.type);
		},
		authRequired() {
			return this.template?.Auth && !this.auth.ok;
		},
		authValuesMissing() {
			console.log("authValuesMissing", this.authValues);
			return this.template?.Auth && Object.values(this.authValues).some((value) => !value);
		},
		authValues() {
			const params = this.template?.Auth?.params ?? [];
			return params.reduce(
				(acc, param) => {
					acc[param] = this.values[param];
					return acc;
				},
				{} as Record<string, any>
			);
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
				} else {
					// For new devices, apply defaults immediately (e.g., default icons based on meter type)
					this.applyDefaults();
				}
			}
		},
		templateName(newValue, oldValue) {
			// Sync back to parent if using externalTemplate
			if (this.externalTemplate !== undefined && newValue !== this.externalTemplate) {
				this.$emit("update:externalTemplate", newValue);
			}

			console.log("templateName changed", { newValue, oldValue });
			// Reset values when template changes (except on initial load or when switching to YAML input)
			// YAML input types set values.type and values.yaml in handleTemplateChange callback
			if (oldValue != null) {
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

			const isYamlInput = this.isYamlInputTypeByValue(newValue as ConfigType);
			if (!isYamlInput) {
				this.loadTemplate();
			}

			this.updateServiceValues();
		},
		usage() {
			// Reload products when usage changes (e.g., meter type selection)
			this.loadProducts();
			// Apply defaults when usage changes (e.g., set default icon for meter type)
			this.applyDefaults();
		},
		externalTemplate(newValue) {
			// Allow parent to control template selection
			if (newValue !== this.templateName) {
				this.templateName = newValue;
			}
		},
		showMainContent(visible) {
			// When main content becomes visible (e.g., meter type selected in MeterModal),
			// apply defaults like icon based on type
			if (visible) {
				this.applyDefaults();
			}
		},
		values: {
			handler() {
				this.test = initialTestState();
				this.updateServiceValues();
			},
			deep: true,
		},
		authValues: {
			handler() {
				if (this.authRequired) {
					this.resetAuthStatus();
				}
			},
			deep: true,
		},
		authRequired() {
			// update on auth state change
			this.updateServiceValues();
		},
		serviceValues: {
			handler(newValue, oldValue) {
				// Apply defaults only for specific params whose service values changed
				Object.keys(newValue).forEach((paramName) => {
					if (!deepEqual(newValue[paramName], oldValue[paramName])) {
						this.applyServiceDefault(paramName);
					}
				});
			},
			deep: true,
		},
	},
	methods: {
		reset() {
			this.values = { ...this.initialValues } as DeviceValues;
			this.test = initialTestState();
			this.resetAuthStatus();
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
					this.values.deviceTitle = device.deviceTitle;
				}
				if (device.deviceIcon !== undefined) {
					this.values.deviceIcon = device.deviceIcon;
				}
				this.applyDefaults();
				this.templateName = this.values.template;

				// Allow parent to handle post-load logic
				if (this.onConfigurationLoaded) {
					this.onConfigurationLoaded(this.values);
				}
				this.checkAuthStatus();
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
				this.checkAuthStatus();
			} catch (e) {
				console.error(e);
			}
			this.loadingTemplate = false;
		},
		resetAuthStatus() {
			this.auth = initialAuthState();
		},
		async checkAuthStatus() {
			this.resetAuthStatus();

			// no auth required
			if (!this.template?.Auth) return;

			// trigger browser validation
			if (this.$refs["form"]) {
				if (!(this.$refs["form"] as HTMLFormElement).reportValidity()) {
					return;
				}
			}

			// validate data
			if (this.authValuesMissing) return;

			const { type } = this.template.Auth;
			const values = this.authValues;
			this.auth.loading = true;
			const result = await this.device.checkAuth(type, values);
			this.auth.loading = false;
			if (result.success) {
				// login already exists
				this.auth.error = null;
				this.auth.ok = true;
			} else if (result.authId) {
				await this.prepareAuthLogin(result.authId);
			} else {
				// something else failed
				this.auth.error = result.error ?? "unknown error";
			}
		},
		async prepareAuthLogin(authId: string) {
			await prepareAuthLogin(this.auth, authId);
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
		handleTemplateChange(e: Event) {
			// ensure this triggers after tempateName watcher
			this.$nextTick(() => {
				if (this.onTemplateChange) {
					this.onTemplateChange(e, this.values);
				}
			});
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
		handleVisibilityChange() {
			this.checkAuthStatus();
		},
		isYamlInputTypeByValue(value: ConfigType): boolean {
			if (this.isYamlInputType) {
				return this.isYamlInputType(value);
			}
			return value === ConfigType.Custom;
		},
		async updateServiceValues() {
			if (this.serviceValuesTimer) {
				clearTimeout(this.serviceValuesTimer);
			}
			this.serviceValuesTimer = setTimeout(async () => {
				this.serviceValues = await fetchServiceValues(this.templateParams, this.values);
			}, 500);
		},
		applyServiceDefault(paramName: string) {
			// Auto-apply single service value when field is empty and required
			const values = this.serviceValues[paramName];
			const param = this.templateParams.find((p) => p.Name === paramName);
			// Only auto-apply if exactly one value is returned, field is empty, and field is required
			if (values?.length === 1 && !this.values[paramName] && param?.Required) {
				this.values[paramName] = values[0];
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
</style>
