<template>
	<DeviceModalBase
		:id="id"
		ref="deviceModalBase"
		modal-id="meterModal"
		device-type="meter"
		:fade="fade"
		:is-sponsor="isSponsor"
		:modal-title="modalTitle"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:is-yaml-input-type="isYamlInput"
		:transform-api-data="transformApiData"
		:filter-template-params="filterTemplateParams"
		:on-template-change="handleTemplateChange"
		:show-main-content="!!meterType"
		:apply-custom-defaults="applyCustomDefaults"
		:custom-fields="customFields"
		:preserve-on-template-change="preserveFields"
		:usage="meterType || undefined"
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
			<div v-if="meterType === 'ext'" class="alert alert-warning mb-4" role="alert">
				<strong>Work in Progress:</strong> This feature is not yet available.
			</div>
		</template>

		<template #before-template="{ values }">
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
import { ConfigType, type SelectedMeterType } from "@/types/evcc";
import type { ModalFade } from "../Helper/GenericModal.vue";
import {
	type DeviceValues,
	type Template,
	type Product,
	type TemplateParam,
	type ApiData,
	type TemplateType,
} from "./DeviceModal";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";

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
	ext: "meter",
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
		type: {
			type: String as () => SelectedMeterType | undefined,
			default: undefined,
		},
		typeChoices: {
			type: Array as () => string[],
			default: () => ["pv", "battery", "aux", "ext"],
		},
		fade: String as PropType<ModalFade>,
		isSponsor: Boolean,
	},
	emits: ["added", "updated", "removed", "close"],
	data() {
		return {
			selectedType: null as string | null,
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
		meterType(): Exclude<TemplateType, "vehicle" | "charger"> | null {
			// @ts-expect-error either this.type or this.selectedType is given
			return this.type || this.selectedType;
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
	},
	methods: {
		selectType(type: string) {
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
					(p.Usages && this.meterType ? p.Usages.includes(this.meterType) : true)
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
				data["usage"] = this.meterType || undefined;
			}
			return data;
		},
		isYamlInput(type: ConfigType): boolean {
			return type === ConfigType.Custom;
		},
		async handleTemplateChange(e: Event, values: DeviceValues) {
			const value = (e.target as HTMLSelectElement).value;
			if (value === ConfigType.Custom) {
				const defaultYaml = await import("./defaultYaml/meter.yaml?raw");
				values.type = ConfigType.Custom;
				values.yaml = defaultYaml.default;
			}
		},
		applyCustomDefaults(_template: Template | null, values: DeviceValues) {
			// Apply default icon when template is loaded
			if (this.meterType && !values["deviceIcon"]) {
				values["deviceIcon"] = defaultIcons[this.meterType] || "";
			}
		},
		handleAdded(name: string) {
			this.$emit("added", this.meterType, name);
		},
		handleRemoved() {
			this.$emit("removed", this.meterType);
		},
		handleClose() {
			this.selectedType = null;
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
