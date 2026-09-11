<template>
	<DeviceModalBase
		:id="id"
		name="thermometer"
		device-type="thermometer"
		:is-sponsor="isSponsor"
		:modal-title="$t(`config.thermometer.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		@added="(name) => $emit('changed', { action: 'added', name })"
		@updated="$emit('changed', { action: 'updated' })"
		@removed="$emit('changed', { action: 'removed' })"
		@disable="$emit('disable', $event)"
	/>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import type { DeviceValues, Product } from "./DeviceModal";
import { type TemplateGroup, customTemplateOption } from "./DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultThermometerYaml from "./defaultYaml/thermometer.yaml?raw";
import { getModal } from "@/configModal";

const initialValues = {
	type: ConfigType.Template,
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};

export default defineComponent({
	name: "ThermometerModal",
	components: {
		DeviceModalBase,
	},
	props: {
		isSponsor: Boolean,
	},
	emits: ["changed", "disable"],
	data() {
		return {
			initialValues,
		};
	},
	computed: {
		id(): number | undefined {
			return getModal("thermometer")?.id;
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
						...products,
						customTemplateOption(this.$t("config.general.customOption")),
					],
				},
			];
		},
		handleTemplateChange(value: string, values: DeviceValues) {
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				values.yaml = defaultThermometerYaml;
			}
		},
	},
});
</script>
