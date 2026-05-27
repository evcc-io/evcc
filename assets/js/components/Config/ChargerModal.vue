<template>
	<DeviceModalBase
		:id="id"
		name="charger"
		device-type="charger"
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
		:hide-template-fields="showOcppOnboarding"
		@added="(name) => emitChanged('added', name)"
		@updated="() => emitChanged('updated')"
		@removed="() => emitChanged('removed')"
		@close="$emit('close')"
		@reset="reset"
	>
		<template v-if="isOcpp" #template-description>
			<p>{{ $t("config.charger.ocppDescription") }}</p>
			<ol class="mb-4">
				<li>{{ $t("config.charger.ocppStep1") }}</li>
				<li>{{ $t("config.charger.ocppStep2") }}</li>
				<li>{{ $t("config.charger.ocppStep3") }}</li>
			</ol>
		</template>

		<template v-if="isOcpp" #after-template-info="{ values }">
			<FormRow
				id="chargerOcppUrl"
				:label="$t('config.charger.ocppLabel')"
				:help="$t('config.charger.ocppHelp', { url: ocppUrlWithStationId })"
			>
				<input
					id="chargerOcppUrl"
					type="text"
					class="form-control border"
					:value="ocppUrl"
					readonly
				/>
			</FormRow>

			<div
				v-if="showOcppOnboarding"
				class="my-4 d-flex justify-content-end gap-2 align-items-center"
			>
				<span v-if="ocppStationIdDetected">{{ $t("config.charger.ocppConnected") }}</span>
				<button
					v-if="ocppStationIdDetected"
					type="button"
					class="btn btn-primary text-nowrap ms-2"
					@click="
						values.stationid = ocppStationIdDetected;
						ocppNextStepConfirmed = true;
					"
				>
					{{ $t("config.charger.ocppNextStep") }}
				</button>
				<button
					v-else
					type="button"
					class="btn btn-outline-primary text-nowrap"
					@click="confirmOcppNextStep"
				>
					<span
						class="spinner-border spinner-border-sm me-2"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.charger.ocppWaiting") }}
				</button>
			</div>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import FormRow from "./FormRow.vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import { ConfigType } from "@/types/evcc";
import { getModal } from "@/configModal";
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
import { LOADPOINT_TYPE, type LoadpointType, type Ocpp } from "@/types/evcc";
import { getOcppUrl, getOcppUrlWithStationId } from "@/utils/ocpp";

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
		ocpp: {
			type: Object as PropType<Ocpp>,
			default: () => ({ config: { port: 0 }, status: { stations: [] } }),
		},
		isSponsor: Boolean,
	},
	emits: ["changed", "close"],
	data() {
		return {
			initialValues,
			customFields: CUSTOM_FIELDS,
			ocppNextStepConfirmed: false,
			currentTemplate: null as Template | null,
			currentValues: {} as DeviceValues,
		};
	},
	computed: {
		id(): number | undefined {
			return getModal("charger")?.id;
		},
		loadpointType(): LoadpointType | null {
			return (getModal("charger")?.type as LoadpointType) || null;
		},
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
		isOcpp(): boolean {
			return (
				this.currentTemplate !== null &&
				this.currentTemplate?.Params.some((p: TemplateParam) => p.Name === "connector") &&
				this.currentTemplate?.Params.some((p: TemplateParam) => p.Name === "stationid")
			);
		},
		ocppUrl(): string | null {
			return this.isOcpp ? getOcppUrl(this.ocpp) : null;
		},
		ocppUrlWithStationId(): string | null {
			return this.isOcpp ? getOcppUrlWithStationId(this.ocpp) : null;
		},
		ocppStationIdDetected(): string | undefined {
			if (!this.isOcpp) {
				return undefined;
			}
			const stations = this.ocpp.status.stations;
			return stations.find((station) => station.status === "unknown")?.id;
		},
		showOcppOnboarding(): boolean {
			if (!this.isOcpp) return false;
			if (this.ocppNextStepConfirmed) return false;
			if (this.currentValues.stationid) return false;
			return true;
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
		confirmOcppNextStep() {
			if (window.confirm(this.$t("config.charger.ocppConfirmContinue"))) {
				this.ocppNextStepConfirmed = true;
			}
		},
		async emitChanged(action: "added" | "updated" | "removed", name?: string) {
			const result = { action, name };
			this.$emit("changed", result);
		},
		reset() {
			this.ocppNextStepConfirmed = false;
			this.currentTemplate = null;
			this.currentValues = {} as DeviceValues;
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
