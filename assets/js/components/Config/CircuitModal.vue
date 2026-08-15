<template>
	<DeviceModalBase
		:id="id"
		name="circuit"
		device-type="circuit"
		:modal-title="$t(`config.circuit.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		@added="$emit('changed', $event)"
		@updated="$emit('changed')"
		@removed="$emit('changed')"
	></DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import type { DeviceValues, Product } from "./DeviceModal";
import {
	type TemplateGroup,
	customTemplateOption,
} from "./DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultCircuitYaml from "./defaultYaml/circuit.yaml?raw";
import { getModal } from "@/configModal";

const initialValues = {
	type: ConfigType.Template,
	template: null,
};

export default defineComponent({
	name: "CircuitModal",
	components: {
		DeviceModalBase,
	},
	emits: ["changed"],
	data() {
		return {
			initialValues,
		};
	},
	computed: {
		id(): number | undefined {
			return getModal("circuit")?.id;
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
						...products.filter(
							(p: Product) => p.group === "generic",
						),
						customTemplateOption(this.$t("config.circuit.custom")),
					],
				},
				{
					label: "primary",
					options: products.filter(
						(p: Product) => p.group !== "generic",
					),
				},
			];
		},
		handleTemplateChange(value: string, values: DeviceValues) {
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				values.yaml = defaultCircuitYaml;
			}
		},
	},
});
</script>
