<template>
	<DeviceModalBase
		:id="id"
		name="circuit"
		device-type="circuit"
		default-template="static"
		:modal-title="$t(`config.circuit.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		:filter-template-params="filterTemplateParams"
		:transform-api-data="transformApiData"
		:preserve-on-template-change="['deviceTitle', 'parent', 'meter']"
		:hide-delete="hasChildren"
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
					:model-value="parentTitle(values.parent)"
					type="String"
					size="w-100"
					class="me-2"
					disabled
				/>
			</FormRow>
		</template>
		<template #before-actions="{ values }">
			<FormRow
				id="circuitParamMeter"
				:class="{ 'mt-4': values.type === ConfigType.Custom }"
				:label="$t('config.circuit.meterLabel')"
				:help="$t('config.circuit.meterHelp')"
				optional
			>
				<PropertyField
					id="circuitParamMeter"
					v-model.trim="values.meter"
					type="String"
					size="w-100"
					class="me-2"
				/>
			</FormRow>
		</template>
		<template v-if="hasChildren" #after-test>
			<p class="evcc-gray">
				{{ $t("config.circuit.deleteChildrenFirst") }}
			</p>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import type { ApiData, DeviceValues, Product, TemplateParam } from "./DeviceModal";
import type { ConfigCircuit } from "@/types/evcc";
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
	props: {
		circuits: { type: Array as PropType<ConfigCircuit[]>, default: () => [] },
	},
	data() {
		return {
			ConfigType,
			parentCircuit: undefined as string | undefined,
		};
	},
	computed: {
		initialValues(): DeviceValues {
			return {
				type: ConfigType.Template,
				template: null,
				parent: this.parentCircuit,
				meter: "",
			};
		},
		id(): number | undefined {
			return getModal("circuit")?.id;
		},
		hasChildren(): boolean | undefined {
			return getModal("circuit")?.hasChildren;
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
					options: [...products, customTemplateOption(this.$t("config.circuit.custom"))],
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
			return params.filter((p) => !["parent", "meter"].includes(p.Name));
		},
		parentTitle(name?: string): string {
			const parent = name ?? this.parentCircuit;
			return (
				this.circuits.find((c: ConfigCircuit) => c.name === parent)?.deviceTitle ||
				parent ||
				""
			);
		},
		transformApiData(data: ApiData): ApiData {
			// always sent, so a parent inside custom yaml cannot break the hierarchy
			data["parent"] = data["parent"] ?? "";
			if (!data["meter"]) delete data["meter"];
			return data;
		},
		handleClose() {
			this.parentCircuit = undefined;
		},
	},
});
</script>
