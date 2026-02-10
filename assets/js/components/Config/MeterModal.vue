<template>
	<DeviceModalBase
		:id="id"
		v-model:external-template="selectedTemplate"
		name="meter"
		device-type="meter"
		:is-sponsor="isSponsor"
		:modal-title="modalTitle"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:transform-api-data="transformApiData"
		:filter-template-params="filterTemplateParams"
		:on-template-change="handleTemplateChange"
		:show-main-content="!!selectedType"
		:apply-custom-defaults="applyCustomDefaults"
		:custom-fields="customFields"
		:preserve-on-template-change="preserveFields"
		:usage="templateUsage"
		:on-configuration-loaded="onConfigurationLoaded"
		@added="(name) => emitChanged('added', name)"
		@updated="() => emitChanged('updated')"
		@removed="() => emitChanged('removed')"
		@close="handleClose"
	>
		<template #pre-content>
			<div v-if="!selectedType" class="d-flex flex-column gap-4">
				<NewDeviceButton
					v-for="t in typeChoices"
					:key="t"
					:title="$t(`config.meter.option.${t}`)"
					class="addButton"
					@click="selectType(t)"
				/>
			</div>
		</template>

		<template #description>
			<p v-if="hasDescription" class="mt-0 mb-4">
				{{ $t(`config.${selectedType}.description`) }}
			</p>
		</template>

		<template #before-template="{ values }">
			<FormRow
				v-if="hasDeviceTitle"
				id="meterParamDeviceTitle"
				:label="$t('config.meter.titleLabel')"
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
			<FormRow
				v-if="hasDeviceIcon"
				id="meterParamDeviceIcon"
				:label="$t('config.icon.label')"
			>
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
			<FormRow
				v-if="selectedType === 'ext'"
				id="meterParamExtMeterUsage"
				:label="$t('config.meter.usage.label')"
			>
				<PropertyField
					id="meterParamExtMeterUsage"
					v-model="extMeterUsage"
					:choice="extMeterUsageOptions"
					:required="!!extMeterUsage"
					:disabled="!isNew"
					@change="extMeterUsageChanged"
				/>
			</FormRow>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import NewDeviceButton from "./NewDeviceButton.vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import { ICONS } from "../VehicleIcon/VehicleIcon.vue";
import { ConfigType, type MeterType, type MeterTemplateUsage } from "@/types/evcc";
import {
	type DeviceValues,
	type Template,
	type Product,
	type TemplateParam,
	type ApiData,
} from "./DeviceModal";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import defaultMeterYaml from "./defaultYaml/meter.yaml?raw";
import { getModal, replaceModal } from "@/configModal";

const initialValues = {
	type: ConfigType.Template,
	deviceTitle: "",
	deviceIcon: "",
	icon: undefined,
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};

const CUSTOM_FIELDS = ["usage", "modbus"];

const defaultIcons: Record<string, string> = {
	aux: "smartconsumer",
	ext: "generic",
};

export default defineComponent({
	name: "MeterModal",
	components: {
		FormRow,
		PropertyField,
		NewDeviceButton,
		DeviceModalBase,
	},
	props: {
		isSponsor: Boolean,
	},
	emits: ["changed", "close"],
	data() {
		return {
			extMeterUsage: "charge" as MeterTemplateUsage,
			selectedTemplate: null as string | null,
			iconChoices: ICONS,
			initialValues,
			customFields: CUSTOM_FIELDS,
			preserveFields: ["deviceTitle", "deviceIcon"],
		};
	},
	computed: {
		id(): number | undefined {
			return getModal("meter")?.id;
		},
		selectedType(): MeterType | undefined {
			return getModal("meter")?.type as MeterType | undefined;
		},
		typeChoices(): MeterType[] {
			return (getModal("meter")?.choices as MeterType[]) || [];
		},
		modalTitle(): string {
			if (this.isNew) {
				if (this.selectedType) {
					return this.$t(`config.${this.selectedType}.titleAdd`);
				} else {
					return this.$t("config.meter.titleChoice");
				}
			}
			return this.$t(`config.${this.selectedType}.titleEdit`);
		},
		templateUsage(): MeterTemplateUsage | undefined {
			if (!this.selectedType) return undefined;

			// For ext meters, the user selects the template usage explicitly
			// For other meter types, the meter type IS the template usage
			if (this.selectedType === "ext") {
				return this.extMeterUsage;
			}
			// For non-ext meters, selectedType directly maps to template usage
			// (grid->grid, pv->pv, battery->battery, charge->charge, aux->aux)
			return this.selectedType;
		},
		hasDeviceTitle(): boolean {
			return ["pv", "battery", "aux", "ext"].includes(this.selectedType || "");
		},
		hasDeviceIcon(): boolean {
			return ["aux", "ext"].includes(this.selectedType || "");
		},
		hasDescription(): boolean {
			return ["ext", "aux"].includes(this.selectedType || "");
		},
		isNew(): boolean {
			return this.id === undefined;
		},
		extMeterUsageOptions() {
			return ["charge", "aux", "grid", "pv", "battery"].map((key) => ({
				name: this.$t(`config.meter.usage.${key}`),
				key,
			}));
		},
	},
	methods: {
		onConfigurationLoaded(values: DeviceValues) {
			// Restore extMeterUsage when editing an existing ext meter
			if (this.selectedType === "ext" && values.usage) {
				this.extMeterUsage = values.usage;
			}
		},
		selectType(type: MeterType) {
			replaceModal("meter", { id: this.id, type });
		},
		provideTemplateOptions(products: Product[]): TemplateGroup[] {
			return [
				{
					label: "generic",
					options: [
						...products.filter((p) => p.group === "generic"),
						customTemplateOption(this.$t("config.general.customOption")),
					],
				},
				{
					label: "specific",
					options: products.filter((p) => p.group !== "generic"),
				},
			];
		},
		filterTemplateParams(params: TemplateParam[]): TemplateParam[] {
			const filtered = params.filter(
				(p) =>
					!CUSTOM_FIELDS.includes(p.Name) &&
					(p.Usages && this.templateUsage
						? p.Usages.includes(this.templateUsage as any)
						: true)
			);

			// Make capacity non-advanced for battery meters
			return filtered.map((p) => {
				if (this.selectedType === "battery" && p.Name === "capacity") {
					p.Advanced = false;
				}
				return p;
			});
		},
		transformApiData(data: ApiData, values: DeviceValues): ApiData {
			if (values.type === ConfigType.Template) {
				// Set the template usage (what the template should do)
				// For ext meters: user-selected usage (grid, pv, battery, charge, aux)
				// For other meters: selectedType itself is the usage
				data.usage = this.templateUsage;
			}
			return data;
		},
		handleTemplateChange(e: Event, values: DeviceValues) {
			const value = (e.target as HTMLSelectElement).value;
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				values.yaml = defaultMeterYaml;
			}
		},
		applyCustomDefaults(_template: Template | null, values: DeviceValues) {
			// Apply default icon when template is loaded or meter type is selected
			if (this.selectedType && !values.deviceIcon) {
				values.deviceIcon = defaultIcons[this.selectedType] || "";
			}
		},
		extMeterUsageChanged() {
			// When ext meter usage changes, reset template selection to force user to reselect
			// This triggers product reload via effectiveUsage computed property change
			this.selectedTemplate = null;
		},
		async emitChanged(action: "added" | "updated" | "removed", name?: string) {
			const type = this.selectedType;
			const result = { action, name, type };
			this.$emit("changed", result);
		},
		handleClose() {
			this.extMeterUsage = "charge";
			this.$emit("close");
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
