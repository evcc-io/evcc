<template>
	<DeviceModalBase
		:id="id"
		modal-id="chargerModal"
		device-type="charger"
		:fade="fade"
		:is-sponsor="isSponsor"
		:modal-title="modalTitle"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:is-type-deprecated="isTypeDeprecated"
		:is-yaml-input-type="isYamlInput"
		:transform-api-data="transformApiData"
		:filter-template-params="filterTemplateParams"
		:on-template-change="handleTemplateChange"
		:apply-custom-defaults="applyCustomDefaults"
		:custom-fields="customFields"
		:get-product-name="getProductName"
		@added="$emit('added', $event)"
		@updated="$emit('updated')"
		@removed="$emit('removed')"
		@close="$emit('close')"
	>
		<template #after-template-info>
			<FormRow
				v-if="ocppUrl"
				id="chargerOcppUrl"
				:label="$t('config.charger.ocppLabel')"
				:help="$t('config.charger.ocppHelp')"
			>
				<input type="text" class="form-control border" :value="ocppUrl" readonly />
			</FormRow>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import FormRow from "./FormRow.vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import { ConfigType } from "@/types/evcc";
import type { ModalFade } from "../Helper/GenericModal.vue";
import {
	type DeviceValues,
	type Template,
	type Product,
	type TemplateParam,
	type ApiData,
	customChargerName,
} from "./DeviceModal";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import customChargerYaml from "./defaultYaml/customCharger.yaml?raw";
import customHeaterYaml from "./defaultYaml/customHeater.yaml?raw";
import heatpumpYaml from "./defaultYaml/heatpump.yaml?raw";
import switchsocketHeaterYaml from "./defaultYaml/switchsocketHeater.yaml?raw";
import switchsocketChargerYaml from "./defaultYaml/switchsocketCharger.yaml?raw";
import sgreadyYaml from "./defaultYaml/sgready.yaml?raw";
import sgreadyRelayYaml from "./defaultYaml/sgreadyRelay.yaml?raw";
import { LOADPOINT_TYPE, type LoadpointType } from "@/types/evcc";

const initialValues = {
	type: ConfigType.Template,
	icon: undefined,
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};

const CUSTOM_FIELDS = ["modbus"];

export default defineComponent({
	name: "ChargerModal",
	components: {
		FormRow,
		DeviceModalBase,
	},
	props: {
		id: Number,
		loadpointType: { type: String as PropType<LoadpointType>, default: null },
		fade: String as PropType<ModalFade>,
		isSponsor: Boolean,
	},
	emits: ["added", "updated", "removed", "close"],
	data() {
		return {
			initialValues,
			customFields: CUSTOM_FIELDS,
			currentTemplate: null as Template | null,
			currentValues: {} as DeviceValues,
		};
	},
	computed: {
		modalTitle(): string {
			if (this.isNew) {
				return this.$t(`config.charger.titleAdd.${this.loadpointType}`);
			}
			return this.$t(`config.charger.titleEdit.${this.loadpointType}`);
		},
		isNew(): boolean {
			return this.id === undefined;
		},
		isHeating(): boolean {
			return this.loadpointType === LOADPOINT_TYPE.HEATING;
		},
		ocppUrl(): string | null {
			const isOcpp =
				this.currentTemplate?.Params.some((p: TemplateParam) => p.Name === "connector") &&
				this.currentTemplate?.Params.some((p: TemplateParam) => p.Name === "stationid");
			if (isOcpp && this.currentValues) {
				return `ws://${window.location.hostname}:8887/${this.currentValues.stationid || ""}`;
			}
			return null;
		},
	},
	methods: {
		provideTemplateOptions(products: Product[]): TemplateGroup[] {
			const result: TemplateGroup[] = [];

			if (this.isHeating) {
				result.push({
					label: "generic",
					options: [
						...products.filter((p) => p.group === "heatinggeneric"),
						...[
							ConfigType.Custom,
							ConfigType.SgReadyRelay,
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
					options: products.filter((p) => p.group === "heating"),
				});
			} else {
				result.push({
					label: "generic",
					options: [
						...products.filter((p) => p.group === "generic"),
						...[ConfigType.Custom, ConfigType.SwitchSocket].map((type) =>
							customTemplateOption(this.$t(customChargerName(type, false)), type)
						),
					],
				});
				result.push({
					label: "chargers",
					options: products.filter((p) => !p.group),
				});
			}

			result.push({
				label: "switchsockets",
				options: products.filter((p) => p.group === "switchsockets"),
			});

			return result;
		},
		filterTemplateParams(params: TemplateParam[]): TemplateParam[] {
			// HACK: soft-require stationid. Can be removed once https://github.com/evcc-io/evcc/pull/22115 is merged
			const filtered = params.map((p) => {
				if (p.Name === "stationid") {
					return { ...p, Required: true };
				}
				return p;
			});

			return filtered.filter(
				(p) =>
					!CUSTOM_FIELDS.includes(p.Name) &&
					(p.Usages ? p.Usages.includes("charger") : true)
			);
		},
		transformApiData(data: ApiData): ApiData {
			if (data.type && this.isYamlInput(data.type)) {
				// Icon is extracted from yaml on GET for UI purpose only. Don't write it back.
				delete data.icon;
			}
			return data;
		},
		isYamlInput(type: ConfigType): boolean {
			return [
				ConfigType.Custom,
				ConfigType.Heatpump,
				ConfigType.SwitchSocket,
				ConfigType.SgReady,
				ConfigType.SgReadyRelay,
				ConfigType.SgReadyBoost, // deprecated
			].includes(type);
		},
		isTypeDeprecated(type: ConfigType): boolean {
			return type === ConfigType.SgReadyBoost;
		},
		handleTemplateChange(e: Event, values: DeviceValues) {
			const value = (e.target as HTMLSelectElement).value as ConfigType;
			if (this.isYamlInput(value)) {
				values.type = value;
				values.yaml = this.defaultYaml(value);
			}
		},
		applyCustomDefaults(template: Template | null, values: DeviceValues) {
			// Store template and values for ocppUrl computation
			this.currentTemplate = template;
			this.currentValues = values;

			if (this.isHeating) {
				// enable heating and integrated device params if exist
				const hasParam = (name: string) => template?.Params.some((p) => p.Name === name);
				["heating", "integrateddevice"].forEach((param) => {
					if (hasParam(param) && values[param] === undefined) {
						values[param] = true;
					}
				});
				// default heater icon
				if (hasParam("icon") && values.icon === undefined) {
					values.icon = "heater";
				}
			}
		},
		defaultYaml(type: ConfigType): string {
			switch (type) {
				case ConfigType.Custom:
					return this.isHeating ? customHeaterYaml : customChargerYaml;
				case ConfigType.Heatpump:
					return heatpumpYaml;
				case ConfigType.SwitchSocket:
					return this.isHeating ? switchsocketHeaterYaml : switchsocketChargerYaml;
				case ConfigType.SgReady:
					return sgreadyYaml;
				case ConfigType.SgReadyRelay:
					return sgreadyRelayYaml;
				default: // template
					return "";
			}
		},
		getProductName(values: DeviceValues, templateName: string | null): string {
			// For YAML input types, return the translated custom name
			if (values.type && this.isYamlInput(values.type)) {
				return this.$t(customChargerName(values.type, this.isHeating));
			}
			// For template types, use the standard logic (deviceProduct or templateName)
			return values.deviceProduct || templateName || "";
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
