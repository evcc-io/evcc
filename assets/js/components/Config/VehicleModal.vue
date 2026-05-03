<template>
	<DeviceModalBase
		:id="id"
		name="vehicle"
		device-type="vehicle"
		:is-sponsor="isSponsor"
		:modal-title="$t(`config.vehicle.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:transform-api-data="transformApiData"
		:filter-template-params="filterTemplateParams"
		:on-template-change="handleTemplateChange"
		default-template="offline"
		@added="$emit('vehicle-changed', $event)"
		@updated="$emit('vehicle-changed')"
		@removed="$emit('vehicle-changed')"
	/>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import { ConfigType } from "@/types/evcc";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import type { Product, ApiData, DeviceValues, TemplateParam } from "./DeviceModal";
import defaultVehicleYaml from "./defaultYaml/vehicle.yaml?raw";
import { getModal } from "@/configModal";

const initialValues = {
	type: ConfigType.Template,
	icon: "car",
	priority: 0,
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};

export default defineComponent({
	name: "VehicleModal",
	components: {
		DeviceModalBase,
	},
	props: {
		isSponsor: Boolean,
	},
	emits: ["vehicle-changed"],
	data() {
		return {
			initialValues,
		};
	},
	computed: {
		id(): number | undefined {
			return getModal("vehicle")?.id;
		},
		isNew(): boolean {
			return this.id === undefined;
		},
	},
	methods: {
		provideTemplateOptions(products: Product[]): TemplateGroup[] {
			return [
				{
					label: "primary",
					options: [
						...products.filter((p: Product) => p.template === "offline"),
						customTemplateOption(this.$t("config.general.customOption")),
					],
				},
				{
					label: "online",
					options: products.filter((p: Product) => !p.group),
				},
				{
					label: "scooter",
					options: products.filter((p: Product) => p.group === "scooter"),
				},
				{
					label: "generic",
					options: products.filter(
						(p: Product) => p.group === "generic" && p.template !== "offline"
					),
				},
			];
		},
		filterTemplateParams(params: TemplateParam[]): TemplateParam[] {
			const result = params.map((p) => {
				if (p.Name === "title" || p.Name === "icon") {
					p.Required = true;
					p.Advanced = false;
				}
				return p;
			});

			result.sort((a, b) => (a.Required ? -1 : 1) - (b.Required ? -1 : 1));
			const order: Record<string, number> = { title: -2, icon: -1 };
			result.sort((a, b) => (order[a.Name] || 0) - (order[b.Name] || 0));

			return result;
		},
		transformApiData(data: ApiData, values: DeviceValues): ApiData {
			if (values.type === ConfigType.Custom) {
				delete data.icon;
				delete data.title;
				delete data.priority;
			}
			if (Array.isArray(data.identifiers)) {
				data.identifiers = data.identifiers.map((i) => i.trim()).filter((i) => i);
			}
			return data;
		},
		handleTemplateChange(e: Event, values: DeviceValues) {
			const value = (e.target as HTMLSelectElement).value;
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				values.yaml = defaultVehicleYaml;
			}
		},
	},
});
</script>
