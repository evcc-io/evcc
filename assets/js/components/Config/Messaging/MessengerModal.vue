<template>
	<DeviceModalBase
		:id="selectedMessengerId"
		fade="right"
		modal-id="messengerModal"
		device-type="messenger"
		:modal-title="$t(`config.messenger.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		@added="$emit('messenger-changed', $event)"
		@updated="$emit('messenger-changed')"
		@removed="$emit('messenger-changed')"
		@close="$emit('messenger-closed')"
	></DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DeviceModalBase from "../DeviceModal/DeviceModalBase.vue";
import type { DeviceValues, Product } from "../DeviceModal";
import { type TemplateGroup, customTemplateOption } from "../DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultMessengerYaml from "../defaultYaml/messenger.yaml?raw";

const initialValues = {
	type: ConfigType.Template,
	template: null,
};

export default defineComponent({
	name: "VehicleModal",
	components: {
		DeviceModalBase,
	},
	props: {
		selectedMessengerId: Number,
	},
	emits: ["messenger-changed", "messenger-closed"],
	data() {
		return {
			initialValues,
		};
	},
	methods: {
		provideTemplateOptions(products: Product[]): TemplateGroup[] {
			return [
				{
					label: "generic",
					options: [customTemplateOption(this.$t("config.general.customOption"))],
				},
				{
					label: "primary",
					options: [...products.filter((p: Product) => p.template !== "offline")],
				},
			];
		},
		handleTemplateChange(e: Event, values: DeviceValues) {
			const value = (e.target as HTMLSelectElement).value;
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				values.yaml = defaultMessengerYaml;
			}
		},
	},
	computed: {
		isNew(): boolean {
			return this.selectedMessengerId === undefined;
		},
	},
});
</script>
