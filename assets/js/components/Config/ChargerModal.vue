<template>
	<GenericModal
		id="chargerModal"
		ref="modal"
		:title="modalTitle"
		data-testid="charger-modal"
		:fade="fade"
		:size="modalSize"
		@open="open"
		@close="close"
	>
		<form ref="form" class="container mx-0 px-0">
			<TemplateSelector
				ref="templateSelect"
				v-model="templateName"
				device-type="charger"
				:is-new="isNew"
				:product-name="productName"
				:groups="templateOptions"
				@change="templateChanged"
			/>

			<YamlEntry
				v-if="showYamlInput"
				v-model="values.yaml"
				type="charger"
				:error-line="test.errorLine"
			/>
			<div v-else>
				<p v-if="loadingTemplate">{{ $t("config.general.templateLoading") }}</p>
				<SponsorTokenRequired v-if="sponsorTokenRequired" />
				<Markdown v-if="description" :markdown="description" class="my-4" />
				<FormRow
					v-if="ocppUrl"
					id="chargerOcppUrl"
					:label="$t('config.charger.ocppLabel')"
					:help="$t('config.charger.ocppHelp')"
				>
					<input type="text" class="form-control border" :value="ocppUrl" readonly />
				</FormRow>

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
					:id="`chargerParam${param.Name}`"
					:key="param.Name"
					v-bind="param"
					v-model="values[param.Name]"
				/>

				<PropertyCollapsible>
					<template v-if="advancedParams.length" #advanced>
						<PropertyEntry
							v-for="param in advancedParams"
							:id="`chargerParam${param.Name}`"
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
				:sponsor-token-required="sponsorTokenRequired"
				@save="isNew ? create() : update()"
				@remove="remove"
				@test="testManually"
			/>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import FormRow from "./FormRow.vue";
import PropertyEntry from "./PropertyEntry.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import Modbus from "./DeviceModal/Modbus.vue";
import DeviceModalActions from "./DeviceModal/Actions.vue";
import GenericModal from "../Helper/GenericModal.vue";
import Markdown from "./Markdown.vue";
import SponsorTokenRequired from "./DeviceModal/SponsorTokenRequired.vue";
import TemplateSelector, { customTemplateOption } from "./DeviceModal/TemplateSelector.vue";
import YamlEntry from "./DeviceModal/YamlEntry.vue";
import { initialTestState, performTest } from "./utils/test";
import { ConfigType } from "@/types/evcc";
import {
	handleError,
	type DeviceValues,
	type Template,
	type Product,
	type ModbusParam,
	type ModbusCapability,
	applyDefaultsFromTemplate,
	createDeviceUtils,
	customChargerName,
} from "./DeviceModal";
import customChargerYaml from "./defaultYaml/customCharger.yaml?raw";
import customHeaterYaml from "./defaultYaml/customHeater.yaml?raw";
import heatpumpYaml from "./defaultYaml/heatpump.yaml?raw";
import switchsocketHeaterYaml from "./defaultYaml/switchsocketHeater.yaml?raw";
import switchsocketChargerYaml from "./defaultYaml/switchsocketCharger.yaml?raw";
import sgreadyYaml from "./defaultYaml/sgready.yaml?raw";
import sgreadyBoostYaml from "./defaultYaml/sgreadyBoost.yaml?raw";
import { LOADPOINT_TYPE, type LoadpointType } from "@/types/evcc";

const initialValues = { type: ConfigType.Template };
const device = createDeviceUtils("charger");

function sleep(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

const CUSTOM_FIELDS = ["modbus"];

type Requirements = {
	Description?: string;
	EVCC?: string[];
};

type ChargerDeviceValues = DeviceValues & {
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
	name: "ChargerModal",
	components: {
		FormRow,
		PropertyEntry,
		GenericModal,
		Modbus,
		PropertyCollapsible,
		Markdown,
		SponsorTokenRequired,
		YamlEntry,
		TemplateSelector,
		DeviceModalActions,
	},
	props: {
		id: Number,
		name: String,
		loadpointType: { type: String as PropType<LoadpointType>, default: null },
		fade: String,
		isSponsor: Boolean,
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
			loadingTemplate: false,
			values: { ...initialValues } as ChargerDeviceValues,
			test: initialTestState(),
		};
	},
	computed: {
		modalTitle() {
			if (this.isNew) {
				return this.$t(`config.charger.titleAdd.${this.loadpointType}`);
			}
			return this.$t(`config.charger.titleEdit.${this.loadpointType}`);
		},
		modalSize() {
			return this.showYamlInput ? "xl" : undefined;
		},
		templateOptions() {
			const result = [];

			if (this.isHeating) {
				result.push({
					label: "generic",
					options: [
						...this.products.filter((p) => p.group === "heatinggeneric"),
						...[
							ConfigType.Custom,
							ConfigType.SgReadyBoost,
							ConfigType.SgReady,
							ConfigType.Heatpump,
							ConfigType.SwitchSocket,
						].map((type) =>
							customTemplateOption(this.$t(customChargerName(type, true)), type)
						),
					],
				});
				result.push({
					label: "heatingdevices",
					options: this.products.filter((p) => p.group === "heating"),
				});
			} else {
				result.push({
					label: "generic",
					options: [
						...this.products.filter((p) => p.group === "generic"),
						...[ConfigType.Custom, ConfigType.SwitchSocket].map((type) =>
							customTemplateOption(this.$t(customChargerName(type, false)), type)
						),
					],
				});
				result.push({
					label: "chargers",
					options: this.products.filter((p) => !p.group),
				});
			}

			result.push({
				label: "switchsockets",
				options: this.products.filter((p) => p.group === "switchsockets"),
			});

			return result;
		},
		templateParams() {
			const params = this.template?.Params || [];
			// HACK: soft-require stationid. Can be removed once https://github.com/evcc-io/evcc/pull/22115 is merged
			params.forEach((p) => {
				if (p.Name === "stationid") p.Required = true;
			});
			return params.filter(
				(p) =>
					!CUSTOM_FIELDS.includes(p.Name) &&
					(p.Usages ? p.Usages.includes("charger") : true)
			);
		},
		normalParams() {
			return this.templateParams.filter((p) => !p.Advanced && !p.Deprecated);
		},
		advancedParams() {
			return this.templateParams.filter((p) => p.Advanced || p.Deprecated);
		},
		modbus() {
			const params = this.template?.Params || [];
			return (params as ModbusParam[]).find((p) => p.Name === "modbus");
		},
		modbusCapabilities() {
			return (this.modbus?.Choice || []) as ModbusCapability[];
		},
		ocppUrl() {
			const isOcpp =
				this.templateParams.some((p) => p.Name === "connector") &&
				this.templateParams.some((p) => p.Name === "stationid");
			if (isOcpp) {
				return `ws://${window.location.hostname}:8887/${this.values["stationid"] || ""}`;
			}
			return null;
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
			return (
				this.values.deviceProduct ||
				this.templateName ||
				this.$t(customChargerName(this.values.type, this.isHeating))
			);
		},
		sponsorTokenRequired() {
			const requirements = this.template?.Requirements as Requirements | undefined;
			return requirements?.EVCC?.includes("sponsorship") && !this.isSponsor;
		},
		apiData() {
			const data: Record<string, any> = {
				...this.modbusDefaults,
				...this.values,
			};
			if (this.values.type === ConfigType.Template) {
				data["template"] = this.templateName;
			}
			return data;
		},
		isNew() {
			return this.id === undefined;
		},
		isHeating() {
			return this.loadpointType === LOADPOINT_TYPE.HEATING;
		},
		isDeletable() {
			return !this.isNew;
		},
		showActions() {
			return this.templateName || this.showYamlInput;
		},
		showYamlInput() {
			return this.isYamlInput(this.values.type);
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				this.templateName = null;
				this.reset();
				this.test = initialTestState();
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
				this.test = initialTestState();
			},
			deep: true,
		},
	},
	methods: {
		reset() {
			this.values = { ...initialValues } as ChargerDeviceValues;
			this.test = initialTestState();
		},
		async loadConfiguration() {
			try {
				const charger = await device.load(this.id as number);
				this.values = charger.config;
				// convert structure to flat list
				// TODO: adjust GET response to match POST/PUT formats
				this.values.type = charger.type;
				this.values.deviceProduct = charger.deviceProduct;
				this.applyDefaults();
				this.templateName = this.values.template;
			} catch (e) {
				console.error(e);
			}
		},
		applyDefaults() {
			applyDefaultsFromTemplate(this.template, this.values);
			if (this.isHeating) {
				// enable heating and integrated device params if exist
				const hasParam = (name: string) =>
					this.template?.Params.some((p) => p.Name === name);
				["heating", "integrateddevice"].forEach((param) => {
					if (hasParam(param) && this.values[param] === undefined) {
						this.values[param] = true;
					}
				});
				// default heater icon
				if (hasParam("icon") && this.values["icon"] === undefined) {
					this.values["icon"] = "heater";
				}
			}
		},
		async loadProducts() {
			if (!this.isModalVisible) {
				return;
			}
			try {
				this.products = await device.loadProducts(this.$i18n?.locale);
			} catch (e) {
				console.error(e);
			}
		},
		async loadTemplate() {
			this.template = null;
			if (!this.templateName || this.isYamlInput(this.templateName as ConfigType)) return;
			this.loadingTemplate = true;
			try {
				this.template = await device.loadTemplate(this.templateName, this.$i18n?.locale);
				this.applyDefaults();
			} catch (e) {
				console.error(e);
			}
			this.loadingTemplate = false;
		},
		async create() {
			// persist selected template product
			if (this.template && this.$refs["templateSelect"]) {
				this.values.deviceProduct = (this.$refs["templateSelect"] as any).getProductName();
			}

			if (this.test.isUnknown) {
				const success = await performTest(
					this.test,
					this.testCharger,
					this.$refs["form"] as HTMLFormElement
				);
				if (!success) return;
				await sleep(100);
			}
			this.saving = true;
			try {
				const { name } = await device.create(this.apiData);
				this.$emit("added", name);
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "create failed");
			}
			this.saving = false;
		},
		async testManually() {
			await performTest(this.test, this.testCharger, this.$refs["form"] as HTMLFormElement);
		},
		async testCharger() {
			return device.test(this.id, this.apiData);
		},
		async update() {
			if (this.test.isUnknown) {
				const success = await performTest(
					this.test,
					this.testCharger,
					this.$refs["form"] as HTMLFormElement
				);
				if (!success) return;
				await sleep(250);
			}
			this.saving = true;
			try {
				await device.update(this.id as number, this.apiData);
				this.$emit("updated");
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await device.remove(this.id as number);
				this.$emit("removed", this.name);
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
		templateChanged(e: Event) {
			const value = (e.target as HTMLSelectElement).value as ConfigType;
			this.reset();
			if (this.isYamlInput(value)) {
				this.values.type = value;
				this.values.yaml = this.defaultYaml(value);
			}
		},
		defaultYaml(type: ConfigType) {
			switch (type) {
				case ConfigType.Custom:
					return this.isHeating ? customHeaterYaml : customChargerYaml;
				case ConfigType.Heatpump:
					return heatpumpYaml;
				case ConfigType.SwitchSocket:
					return this.isHeating ? switchsocketHeaterYaml : switchsocketChargerYaml;
				case ConfigType.SgReady:
					return sgreadyYaml;
				case ConfigType.SgReadyBoost:
					return sgreadyBoostYaml;
				default: // template
					return "";
			}
		},
		isYamlInput(type: ConfigType) {
			return [
				ConfigType.Custom,
				ConfigType.Heatpump,
				ConfigType.SwitchSocket,
				ConfigType.SgReady,
				ConfigType.SgReadyBoost,
			].includes(type);
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
