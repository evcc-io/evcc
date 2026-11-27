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
		hide-disable
		@reset="resetMeterSelection"
		@added="$emit('changed', $event)"
		@updated="$emit('changed')"
		@removed="$emit('changed')"
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
			<div :class="{ 'mt-4': values.type === ConfigType.Custom }">
				<FormRow
					v-if="!hasParentCircuit"
					id="circuitParamMeterSelection"
					data-testid="circuit-meter-selection"
					:label="$t('config.circuit.meterSelectionLabel')"
					:help="meterSelectionHelp(values)"
				>
					<PropertyField
						id="circuitParamMeterSelection"
						:model-value="meterSelection(values)"
						@update:model-value="meterSelectionChanged($event, values)"
						type="Choice"
						size="w-100"
						class="me-2"
						:choice="meterSelectionOptions"
						required
					/>
					<template v-if="meterSelection(values) === 'none'" #footer>
						<div class="text-warning mt-1" role="status">
							{{ $t("config.circuit.meterSelectionWarningNoMeter") }}
						</div>
					</template>
				</FormRow>
				<FormRow
					v-if="hasParentCircuit || meterSelection(values) === 'dedicated'"
					id="circuitParamMeter"
					:label="$t('config.circuit.meterLabel')"
					:help="$t('config.circuit.meterHelp')"
					:optional="hasParentCircuit"
				>
					<DeviceRefBox
						v-if="values.meter"
						:title="meterTitle(meters, values.meter)"
						compact
						@edit="createMeter(values)"
					/>
					<button
						v-else
						type="button"
						class="d-flex btn btn-sm align-items-center gap-2 mb-3 btn-outline-secondary border-0 evcc-gray"
						data-testid="circuit-meter-change"
						tabindex="0"
						@click="createMeter(values)"
					>
						<shopicon-regular-plus
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-plus>
						{{ $t("config.circuit.addMeter") }}
					</button>
				</FormRow>
			</div>
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
import { meterTitle } from "@/utils/circuits.ts";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import DeviceRefBox from "./DeviceRefBox.vue";

enum MeterSelection {
	NONE = "none",
	GRID = "grid",
	DEDICATED = "dedicated",
}

export default defineComponent({
	name: "CircuitModal",
	components: {
		DeviceModalBase,
		FormRow,
		PropertyField,
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
			selectedMeter: null as MeterSelection | null,
		};
	},
	computed: {
		getParentCircuit(): string | undefined {
			const parentId = getModal("circuit")?.parent;
			if (parentId) return this.circuits.find((c) => c.id === parentId)?.name;
			const parent = this.circuits.find((c) => c.id === this.id)?.config.parent;
			return parent ? String(parent) : undefined;
		},
		hasParentCircuit(): boolean {
			return !!this.getParentCircuit;
		},
		initialValues(): DeviceValues {
			return {
				type: ConfigType.Template,
				template: null,
				parent: this.getParentCircuit,
				// new root circuits measure the whole installation, not just the charging points
				meter: (this.isNew && !this.hasParentCircuit && this.gridMeter?.name) || "",
			};
		},
		id(): number | undefined {
			return getModal("circuit")?.id;
		},
		hasChildren(): boolean {
			const name = this.circuits.find((c) => c.id === this.id)?.name;
			return !!name && this.circuits.some((c) => c.config.parent === name);
		},
		isNew(): boolean {
			return this.id === undefined;
		},
		meterSelectionOptions() {
			const options = [
				{ key: MeterSelection.NONE, name: this.$t("config.circuit.meterNone") },
			];

			if (!this.hasParentCircuit && this.gridMeter) {
				options.push({
					key: MeterSelection.GRID,
					name: this.$t("config.circuit.meterGrid"),
				});
			}
			options.push({
				key: MeterSelection.DEDICATED,
				name: this.$t("config.circuit.meterDedicated"),
			});

			return options;
		},
	},
	methods: {
		meterTitle,
		// the meter reference wins, the user selection covers dedicated without a meter yet
		meterSelection(values: { meter?: string }): MeterSelection {
			if (!values.meter) return this.selectedMeter ?? MeterSelection.NONE;
			return values.meter === this.gridMeter?.name
				? MeterSelection.GRID
				: MeterSelection.DEDICATED;
		},
		meterSelectionHelp(values: { meter?: string }): string {
			const key = { none: "NoMeter", grid: "GridMeter", dedicated: "DedicatedMeter" };
			return this.$t(`config.circuit.meterSelectionHelp${key[this.meterSelection(values)]}`);
		},
		meterSelectionChanged(selection: MeterSelection, values: { meter?: string }) {
			this.selectedMeter = selection;
			if (selection === MeterSelection.GRID) {
				values.meter = this.gridMeter?.name;
			} else {
				delete values.meter;
			}
		},
		resetMeterSelection() {
			this.selectedMeter = null;
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
		parentTitle(name: string): string {
			return this.circuits.find((c) => c.name === name)?.deviceTitle || name;
		},
		transformApiData(data: ApiData): ApiData {
			// always sent, so a parent inside custom yaml cannot break the hierarchy
			data["parent"] = data["parent"] ?? "";
			if (!data["meter"]) delete data["meter"];
			return data;
		},
		async createMeter(values: { meter?: string }) {
			const meter = this.meters.find((m) => m.name === values.meter);
			const result = await openModal("meter", { id: meter?.id, type: "circuit" });
			if (result.action === "added" && result.name) {
				values.meter = result.name;
			} else if (result.action === "removed") {
				delete values.meter;
			}
		},
	},
});
</script>
