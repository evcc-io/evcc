<template>
	<DeviceModalBase
		:id="id"
		ref="base"
		name="messenger"
		device-type="messenger"
		:modal-title="$t(`config.messenger.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		@added="onAdded($event)"
		@updated="$emit('changed')"
		@removed="onRemoved"
	></DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DeviceModalBase from "../DeviceModal/DeviceModalBase.vue";
import type { DeviceValues, Product } from "../DeviceModal";
import { type TemplateGroup, customTemplateOption } from "../DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultMessengerYaml from "../defaultYaml/messenger.yaml?raw";
import { getModal } from "@/configModal";
import { registerServiceWorker, unsubscribeCurrentBrowser } from "@/utils/push";

const PUSH_TEMPLATE = "webpush";

const initialValues = {
	type: ConfigType.Template,
	template: null,
};

export default defineComponent({
	name: "MessengerModal",
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
			return getModal("messenger")?.id;
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
						customTemplateOption(this.$t("config.messenger.custom")),
					],
				},
				{
					label: "primary",
					options: [...products.filter((p: Product) => p.group !== "generic")],
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
		async onAdded(event: unknown) {
			const base = this.$refs["base"] as unknown as InstanceType<typeof DeviceModalBase>;
			if (base?.templateName === PUSH_TEMPLATE) {
				await registerServiceWorker();
			}
			this.$emit("changed", event);
		},
		async onRemoved() {
			const base = this.$refs["base"] as unknown as InstanceType<typeof DeviceModalBase>;
			if (base?.templateName === PUSH_TEMPLATE) {
				await unsubscribeCurrentBrowser();
			}
			this.$emit("changed");
		},
	},
});
</script>
