<template>
	<DeviceModalBase
		:id="id"
		modal-id="messengerModal"
		device-type="messenger"
		:modal-title="$t(`config.messenger.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		default-template="email"
		@added="$emit('messenger-changed', $event)"
		@updated="$emit('messenger-changed')"
		@removed="$emit('messenger-changed')"
	/>
</template>

<script lang="ts">
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import type { DeviceValues, Product } from "./DeviceModal";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultMessengerYaml from "./defaultYaml/messenger.yaml?raw";

const initialValues = {
	type: ConfigType.Template,
	template: null,
};

export default {
	name: "MessengerModal",
	components: { DeviceModalBase },
	props: {
		id: Number,
	},
	emits: ["messenger-changed"],
	data() {
		return {
			initialValues,
		};
	},
	computed: {
		isNew(): boolean {
			return this.id === undefined;
		},
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
};
</script>
