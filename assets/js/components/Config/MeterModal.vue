<template>
	<DeviceModalBase
		:id="id"
		v-model:external-template="selectedTemplate"
		modal-id="meterModal"
		device-type="meter"
		:fade="fade"
		:is-sponsor="isSponsor"
		:modal-title="modalTitle"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:transform-api-data="transformApiData"
		:filter-template-params="filterTemplateParams"
		:on-template-change="handleTemplateChange"
		:show-main-content="!!meterType"
		:apply-custom-defaults="applyCustomDefaults"
		:custom-fields="customFields"
		:preserve-on-template-change="preserveFields"
		:usage="templateUsage"
		:on-configuration-loaded="onConfigurationLoaded"
		@added="handleAdded"
		@updated="$emit('updated')"
		@removed="handleRemoved"
		@close="handleClose"
	>
		<template #pre-content>
			<div v-if="!meterType" class="d-flex flex-column gap-4">
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
				{{ $t(`config.${meterType}.description`) }}
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
				v-if="meterType === 'ext'"
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
import { defineComponent, type PropType } from "vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import NewDeviceButton from "./NewDeviceButton.vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import { ICONS } from "../VehicleIcon/VehicleIcon.vue";
import { ConfigType, type MeterType, type MeterTemplateUsage } from "@/types/evcc";
import type { ModalFade } from "../Helper/GenericModal.vue";
import {
	type DeviceValues,
	type Template,
	type Product,
	type TemplateParam,
	type ApiData,
} from "./DeviceModal";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import defaultMeterYaml from "./defaultYaml/meter.yaml?raw";

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
		id: Number,
		type: { type: String as PropType<MeterType>, default: null },
		typeChoices: { type: Array as () => MeterType[], default: () => [] },
		fade: String as PropType<ModalFade>,
		isSponsor: Boolean,
	},
	emits: ["added", "updated", "removed", "close"],
	data() {
		return {
			selectedType: null as MeterType | null,
			extMeterUsage: "charge" as MeterTemplateUsage,
			selectedTemplate: null as string | null,
			iconChoices: ICONS,
			initialValues,
			customFields: CUSTOM_FIELDS,
			preserveFields: ["deviceTitle", "deviceIcon"],
		};
	},
	computed: {
		modalTitle(): string {
			if (this.isNew) {
				if (this.meterType) {
					return this.$t(`config.${this.meterType}.titleAdd`);
				} else {
					return this.$t("config.meter.titleChoice");
				}
			}
			return this.$t(`config.${this.meterType}.titleEdit`);
		},
		meterType(): MeterType | null {
			return this.type || this.selectedType;
		},
		templateUsage(): MeterTemplateUsage | undefined {
			if (!this.meterType) return undefined;

			// For ext meters, the user selects the template usage explicitly
			// For other meter types, the meter type IS the template usage
			if (this.meterType === "ext") {
				return this.extMeterUsage;
			}
			// For non-ext meters, meterType directly maps to template usage
			// (grid->grid, pv->pv, battery->battery, charge->charge, aux->aux)
			return this.meterType;
		},
		hasDeviceTitle(): boolean {
			return ["pv", "battery", "aux", "ext"].includes(this.meterType || "");
		},
		hasDeviceIcon(): boolean {
			return ["aux", "ext"].includes(this.meterType || "");
		},
		hasDescription(): boolean {
			return ["ext", "aux"].includes(this.meterType || "");
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
			if (this.meterType === "ext" && values.usage) {
				this.extMeterUsage = values.usage;
			}
		},
		selectType(type: MeterType) {
			this.selectedType = type;
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
				if (this.meterType === "battery" && p.Name === "capacity") {
					p.Advanced = false;
				}
				return p;
			});
		},
		transformApiData(data: ApiData, values: DeviceValues): ApiData {
			if (values.type === ConfigType.Template) {
				// Set the template usage (what the template should do)
				// For ext meters: user-selected usage (grid, pv, battery, charge, aux)
				// For other meters: meterType itself is the usage
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
			if (this.meterType && !values.deviceIcon) {
				values.deviceIcon = defaultIcons[this.meterType] || "";
			}
		},
		extMeterUsageChanged() {
			// When ext meter usage changes, reset template selection to force user to reselect
			// This triggers product reload via effectiveUsage computed property change
			this.selectedTemplate = null;
		},
		handleAdded(name: string) {
			this.$emit("added", this.meterType, name);
		},
		handleRemoved() {
			this.$emit("removed", this.meterType);
		},
		handleClose() {
			this.selectedType = null;
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
