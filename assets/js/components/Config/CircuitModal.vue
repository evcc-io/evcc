<template>
	<DeviceModalBase
		:id="id"
		name="circuit"
		device-type="circuit"
		:modal-title="$t(`config.circuit.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		:filter-template-params="filterTemplateParams"
		@added="$emit('changed', $event)"
		@updated="$emit('changed')"
		@removed="$emit('changed')"
		@close="handleClose"
	>
		<template #before-template="{ values }">
			<FormRow id="circuitParamDeviceTitle" :label="$t('config.circuit.titleLabel')">
				<PropertyField
					id="circuitParamDeviceTitle"
					v-model.trim="values.deviceTitle"
					type="String"
					size="w-100"
					class="me-2"
					required
				/>
			</FormRow>
			<FormRow
				v-if="values.parent"
				id="circuitParamDeviceParentCircuit"
				:label="$t('config.circuit.parentCircuit')"
			>
				<PropertyField
					id="circuitParamDeviceParentCircuit"
					:model-value.trim="values.parent ?? parentCircuit"
					type="String"
					size="w-100"
					class="me-2"
					disabled
				/>
			</FormRow>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import type { DeviceValues, Product, TemplateParam } from "./DeviceModal";
import { type TemplateGroup, customTemplateOption } from "./DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultCircuitYaml from "./defaultYaml/circuit.yaml?raw";
import { getModal } from "@/configModal";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";

export default defineComponent({
	name: "CircuitModal",
	components: {
		DeviceModalBase,
		FormRow,
		PropertyField,
	},
	emits: ["changed"],
	data() {
		return {
			parentCircuit: undefined as string | undefined,
		};
	},
	computed: {
		initialValues(): DeviceValues {
			return {
				type: ConfigType.Template,
				template: null,
				parent: this.parentCircuit,
			};
		},
		id(): number | undefined {
			return getModal("circuit")?.id;
		},
		isNew(): boolean {
			return this.id === undefined;
		},
	},
	methods: {
		setParentCircuit(parentCircuit?: string) {
			this.parentCircuit = parentCircuit;
		},
		provideTemplateOptions(products: Product[]): TemplateGroup[] {
			return [
				{
					label: "generic",
					options: [
						...products.filter((p: Product) => p.group === "generic"),
						customTemplateOption(this.$t("config.circuit.custom")),
					],
				},
				{
					label: "primary",
					options: products.filter((p: Product) => p.group !== "generic"),
				},
			];
		},
		handleTemplateChange(value: string, values: DeviceValues) {
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				values.yaml = defaultCircuitYaml;
			}
		},
		filterTemplateParams(params: TemplateParam[]): TemplateParam[] {
			return params.filter((p) => p.Name !== "parent");
		},
		handleClose() {
			this.parentCircuit = undefined;
		},
	},
});
</script>
