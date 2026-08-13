<template>
	<DeviceModalBase
		:id="id"
		name="curtailer"
		device-type="curtailer"
		:modal-title="$t(`config.curtailer.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		:preserve-on-template-change="preserveFields"
		@added="(name) => $emit('changed', { action: 'added', name })"
		@updated="$emit('changed', { action: 'updated' })"
		@removed="$emit('changed', { action: 'removed' })"
	>
		<template #description>
			<p class="mt-0 mb-4">{{ $t("config.curtailer.description") }}</p>
		</template>

		<template #before-template="{ values }">
			<FormRow id="curtailerParamDeviceTitle" :label="$t('config.general.title')">
				<PropertyField
					id="curtailerParamDeviceTitle"
					v-model.trim="values.deviceTitle"
					type="String"
					size="w-100"
					class="me-2"
					required
				/>
			</FormRow>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import type { DeviceValues, Product } from "./DeviceModal";
import { type TemplateGroup, customTemplateOption } from "./DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultCurtailerYaml from "./defaultYaml/curtailer.yaml?raw";
import { getModal } from "@/configModal";

const initialValues = {
	type: ConfigType.Template,
	deviceTitle: "",
	template: null,
};

export default defineComponent({
	name: "CurtailerModal",
	components: {
		DeviceModalBase,
		FormRow,
		PropertyField,
	},
	emits: ["changed"],
	data() {
		return {
			initialValues,
			preserveFields: ["deviceTitle"],
		};
	},
	computed: {
		id(): number | undefined {
			return getModal("curtailer")?.id;
		},
		isNew(): boolean {
			return this.id === undefined;
		},
	},
	methods: {
		provideTemplateOptions(products: Product[]): TemplateGroup[] {
			return [
				{
					label: "generic",
					options: [
						...products.filter((p: Product) => p.group === "generic"),
						customTemplateOption(this.$t("config.general.customOption")),
					],
				},
				{
					label: "specific",
					options: products.filter((p: Product) => p.group !== "generic"),
				},
			];
		},
		handleTemplateChange(value: string, values: DeviceValues) {
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				values.yaml = defaultCurtailerYaml;
			}
		},
	},
});
</script>
