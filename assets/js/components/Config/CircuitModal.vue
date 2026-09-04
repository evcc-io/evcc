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
		:on-configuration-loaded="handleConfigurationLoaded"
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
			<div v-if="values.type !== ConfigType.Custom">
				<FormRow
					v-if="parentCircuit === undefined"
					id="circuitParamMeterSelection"
					data-testid="circuit-meter-selection"
					:label="$t('config.circuit.meterSelectionLabel')"
					:help="meterSelectionHelp"
				>
					<PropertyField
						id="circuitParamMeterSelection"
						v-model:model-value="meterSelection"
						@update:model-value="meterSelectionChanged($event, values)"
						type="Choice"
						size="w-100"
						class="me-2"
						:choice="meterSelectionOptions"
						required
					/>
				</FormRow>
				<FormRow
					v-if="parentCircuit !== undefined || meterSelection === 'dedicated'"
					id="circuitParamMeter"
					:label="$t('config.circuit.meterLabel')"
					:help="$t('config.circuit.meterHelp')"
					:optional="parentCircuit !== undefined"
				>
					<DeviceRefBox
						v-if="values.meter && meterSelection !== 'grid'"
						:title="getMeterTitle(values.meter)"
						compact
						@edit="changeMeter(values)"
					/>
					<button
						v-else
						type="button"
						class="d-flex btn btn-sm align-items-center gap-2 mb-3 btn-outline-secondary border-0 evcc-gray"
						data-testid="circuit-meter-change"
						tabindex="0"
						@click="changeMeter(values)"
					>
						<shopicon-regular-plus
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-plus>
						{{ $t("config.circuit.addMeter") }}
					</button>
				</FormRow>
			</div>
			<ChangeMeterModal :meters="meters" :meter-id="values.meter" />
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
import type { ConfigCircuit, ConfigMeter } from "@/types/evcc";
import { type TemplateGroup, customTemplateOption } from "./DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultCircuitYaml from "./defaultYaml/circuit.yaml?raw";
import { getModal, openModal } from "@/configModal";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import ChangeMeterModal from "./ChangeMeterModal.vue";
import DeviceRefBox from "./DeviceRefBox.vue";

export default defineComponent({
	name: "CircuitModal",
	components: {
		DeviceModalBase,
		FormRow,
		PropertyField,
		ChangeMeterModal,
		DeviceRefBox,
	},
	emits: ["changed"],
	props: {
		circuits: {
			type: Array as PropType<ConfigCircuit[]>,
			default: () => [],
		},
		meters: {
			type: Array as PropType<ConfigMeter[]>,
			default: () => [],
		},
		gridMeter: { type: Object as PropType<ConfigMeter> },
	},
	data() {
		return {
			ConfigType,
			parentCircuit: undefined as string | undefined,
			meterSelection: "none",
		};
	},
	computed: {
		getMeterTitle() {
			return (name: string) => {
				const meters = this.meters.filter((m) => m.name === name);
				if (meters.length === 1) {
					return meters[0].deviceTitle;
				}
				return "";
			};
		},
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
		meterSelectionHelp() {
			switch (this.meterSelection) {
				case "none":
					return this.$t("config.circuit.meterSelectionHelpNoMeter");
				case "grid":
					return this.$t("config.circuit.meterSelectionHelpGridMeter");
				case "dedicated":
					return this.$t("config.circuit.meterSelectionHelpDedicatedMeter");
				default:
					return "";
			}
		},
		meterSelectionOptions() {
			const options = [{ key: "none", name: this.$t("config.circuit.meterNone") }];

			if (this.gridMeter) {
				options.push({
					key: "grid",
					name: this.$t("config.circuit.meterGrid"),
				});
			}
			options.push({
				key: "dedicated",
				name: this.$t("config.circuit.meterDedicated"),
			});

			return options;
		},
	},
	methods: {
		meterSelectionChanged(selection: string, values: { meter?: string }) {
			if (selection === "grid") {
				values.meter = this.gridMeter?.name;
			} else if (selection === "none") {
				delete values.meter;
			} else if (values.meter === this.gridMeter?.name) {
				delete values.meter;
			}
		},
		handleConfigurationLoaded(values: DeviceValues) {
			if (!values["meter"]) {
				this.meterSelection = "none";
			} else if (this.gridMeter && values["meter"] === this.gridMeter.name) {
				this.meterSelection = "grid";
			} else {
				this.meterSelection = "dedicated";
			}
		},
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
		async changeMeter(values: { meter?: string }) {
			const meter = this.meters.find((m) => m.name === values.meter);
			const result = await openModal("changeMeter", { id: meter?.id });
			if (result.action === "added" && result.name) {
				this.meterSelection =
					this.gridMeter && result.name === this.gridMeter.name ? "grid" : "dedicated";
				values.meter = result.name;
			} else if (result.action === "removed") {
				this.meterSelection = "none";
				delete values.meter;
			}
		},
	},
});
</script>
