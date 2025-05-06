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
				v-if="isNew"
				ref="templateSelect"
				v-model="templateName"
				device-type="charger"
				:is-new="isNew"
				:product-name="productName"
				:groups="templateOptions"
				@change="templateChanged"
			/>

			<YamlEntry v-if="values.type === 'custom'" v-model="values.yaml" type="charger" />
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
import { defineComponent } from "vue";
import FormRow from "./FormRow.vue";
import PropertyEntry from "./PropertyEntry.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import api from "@/api";
import Modbus from "./DeviceModal/Modbus.vue";
import DeviceModalActions from "./DeviceModal/Actions.vue";
import GenericModal from "../Helper/GenericModal.vue";
import Markdown from "./Markdown.vue";
import SponsorTokenRequired from "./DeviceModal/SponsorTokenRequired.vue";
import TemplateSelector from "./DeviceModal/TemplateSelector.vue";
import YamlEntry from "./DeviceModal/YamlEntry.vue";
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
import defaultYaml from "./defaultYaml/charger.yaml?raw";

const initialValues = { type: ConfigType.Template };

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
				return this.$t(`config.charger.titleAdd`);
			}
			return this.$t(`config.charger.titleEdit`);
		},
		modalSize() {
			return this.values.type === ConfigType.Custom ? "xl" : undefined;
		},
		genericOptions() {
			return this.products.filter((p) => p.group === "generic");
		},
		chargerOptions() {
			return this.products.filter((p) => !p.group);
		},
		switchSocketOptions() {
			return this.products.filter((p) => p.group === "switchsockets");
		},
		heatingdevicesOptions() {
			return this.products.filter((p) => p.group === "heating");
		},
		templateOptions() {
			return [
				{
					label: "generic",
					options: this.genericOptions,
				},
				{
					label: "chargers",
					options: this.chargerOptions,
				},
				{
					label: "switchsockets",
					options: this.switchSocketOptions,
				},
				{
					label: "heatingdevices",
					options: this.heatingdevicesOptions,
				},
			];
		},
		templateParams() {
			const params = this.template?.Params || [];
			return params.filter((p) => !CUSTOM_FIELDS.includes(p.Name));
		},
		normalParams() {
			return this.templateParams.filter((p) => !p.Advanced);
		},
		advancedParams() {
			return this.templateParams.filter((p) => p.Advanced);
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
				return `ws://${window.location.hostname}:8887`;
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
			return this.values.deviceProduct || this.templateName || "";
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
		isDeletable() {
			return !this.isNew;
		},
		showActions() {
			return this.templateName || this.values.type === ConfigType.Custom;
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
				const charger = (await api.get(`config/devices/charger/${this.id}`)).data.result;
				this.values = charger.config;
				// convert structure to flat list
				// TODO: adjust GET response to match POST/PUT formats
				this.values.type = charger.type;
				this.values.deviceProduct = charger.deviceProduct;
				this.applyDefaultsFromTemplate();
				this.templateName = this.values.template;
			} catch (e) {
				console.error(e);
			}
		},
		async loadProducts() {
			if (!this.isModalVisible) {
				return;
			}
			try {
				const opts = { params: { lang: this.$i18n?.locale } };
				this.products = (await api.get("config/products/charger", opts)).data.result;
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
				const result = await api.get("config/templates/charger", opts);
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
					this.testCharger,
					this.$refs["form"] as HTMLFormElement
				);
				if (!success) return;
				await sleep(100);
			}
			this.saving = true;
			try {
				const response = await api.post("config/devices/charger", this.apiData);
				const { name } = response.data.result;
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
			let url = "config/test/charger";
			if (!this.isNew) {
				url += `/merge/${this.id}`;
			}
			return await api.post(url, this.apiData, { timeout });
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
				await api.put(`config/devices/charger/${this.id}`, this.apiData);
				this.$emit("updated");
				(this.$refs["modal"] as any).close();
			} catch (e) {
				handleError(e, "update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await api.delete(`config/devices/charger/${this.id}`);
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
		templateChanged() {
			this.reset();
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
