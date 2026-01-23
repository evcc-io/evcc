<template>
	<DeviceModalBase
		:id="id"
		modal-id="messengerModal"
		device-type="messenger"
		:modal-title="$t(`config.messenger.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		default-template="email"
		@added="$emit('messenger-changed', $event)"
		@updated="$emit('messenger-changed')"
		@removed="$emit('messenger-changed')"
	/>
</template>

<script lang="ts">
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import type { Product } from "./DeviceModal";
import type { TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";

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
			console.log(products);
			return [];
		},
	},
};
</script>
