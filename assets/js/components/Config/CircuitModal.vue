<template>
	<DeviceModalBase
		:id="id"
		name="circuit"
		device-type="circuit"
		default-template="static"
		size="xl"
		:modal-title="$t(`config.circuit.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:on-template-change="handleTemplateChange"
		:filter-template-params="filterTemplateParams"
		:transform-api-data="transformApiData"
		:preserve-on-template-change="['deviceTitle', 'parent', 'meter']"
		:on-configuration-loaded="handleConfigurationLoaded"
		:hide-delete="hasChildren"
		hide-disable
		@added="handleAdded"
		@updated="handleUpdated"
		@removed="handleRemoved"
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
					v-if="hasParentCircuit || meterSelection === 'dedicated'"
					id="circuitParamMeter"
					:label="$t('config.circuit.meterLabel')"
					:help="$t('config.circuit.meterHelp')"
					:optional="hasParentCircuit"
				>
					<DeviceRefBox
						v-if="values.meter && meterSelection !== 'grid'"
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
				<FormRow
					id="circuitParamLoadpoint"
					:label="$t('config.circuit.loadpointLabel')"
					:help="$t('config.circuit.loadpointHelp')"
				>
					<MultiSelect
						id="circuitParamLoadpoint"
						v-model="selectedLoadpointIds"
						:options="loadpointOptions"
						:disabled="loadpointOptions.length === 0"
					>
						{{ loadpointsLabel }}
					</MultiSelect>
					<template #additional-help>
						<div v-if="assignedLoadpoints.length" class="text-gray hyphenate">
							{{ $t("config.circuit.assignedLoadpoints") }}
							<code class="ms-1">
								{{ assignedLoadpoints }}
							</code>
						</div>
						<div v-if="yamlLoadpoints.length" class="text-gray hyphenate">
							{{ $t("config.circuit.yamlLoadpoints") }}
							<code class="ms-1">
								{{ yamlLoadpoints }}
							</code>
						</div>
					</template>
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
import type { ConfigCircuit, ConfigLoadpoint, ConfigMeter } from "@/types/evcc";
import { type TemplateGroup, customTemplateOption } from "./DeviceModal/TemplateSelector.vue";
import { ConfigType } from "@/types/evcc";
import defaultCircuitYaml from "./defaultYaml/circuit.yaml?raw";
import { getModal, openModal } from "@/configModal";
import { meterTitle } from "@/utils/circuits.ts";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import DeviceRefBox from "./DeviceRefBox.vue";
import MultiSelect from "../Helper/MultiSelect.vue";
import api from "@/api.ts";

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
		MultiSelect,
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
		loadpoints: {
			type: Array as PropType<ConfigLoadpoint[]>,
			default: () => [],
		},
		gridMeter: { type: Object as PropType<ConfigMeter> },
	},
	data() {
		return {
			ConfigType,
			meterSelection: MeterSelection.NONE,
			selectedLoadpointIds: [] as number[],
		};
	},
	computed: {
		loadpointsLabel() {
			if (this.loadpointOptions.length === 0)
				return this.$t("config.circuit.noLoadpointsAssignable");

			const loadpoints = this.availableLoadpoints
				.filter((l) => l.id && this.selectedLoadpointIds.includes(l.id))
				.map((l) => l.title);

			if (loadpoints.length === 0) return this.$t("config.circuit.noLoadpointsAssigned");
			return loadpoints.join(", ");
		},
		loadpointOptions() {
			const availableLoadpoints = this.availableLoadpoints.map((l) => ({
				name: l.title,
				value: l.id!,
			}));

			return availableLoadpoints;
		},
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
				meter: "",
			};
		},
		id(): number | undefined {
			return getModal("circuit")?.id;
		},
		circuitName(): string | undefined {
			return this.circuits.find((circuit) => circuit.id === this.id)?.name;
		},
		hasChildren(): boolean {
			const name = this.circuits.find((c) => c.id === this.id)?.name;
			return !!name && this.circuits.some((c) => c.config.parent === name);
		},
		isNew(): boolean {
			return this.id === undefined;
		},
		meterSelectionHelp() {
			switch (this.meterSelection) {
				case MeterSelection.NONE:
					return this.$t("config.circuit.meterSelectionHelpNoMeter");
				case MeterSelection.GRID:
					return this.$t("config.circuit.meterSelectionHelpGridMeter");
				case MeterSelection.DEDICATED:
					return this.$t("config.circuit.meterSelectionHelpDedicatedMeter");
				default:
					return "";
			}
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
		availableLoadpoints() {
			return this.loadpoints.filter(
				(l) => l.id && (l.circuit === undefined || l.circuit === `db:${this.id}`)
			);
		},
		assignedLoadpoints() {
			const lps = this.loadpoints.filter(
				(l) => l.id && l.circuit && l.circuit !== `db:${this.id}`
			);
			return this.formatLoadpoints(lps);
		},
		yamlLoadpoints() {
			const lps = this.loadpoints.filter((l) => !l.id);
			return this.formatLoadpoints(lps);
		},
	},
	watch: {
		id: {
			immediate: true,
			handler(newId: number | undefined) {
				this.selectedLoadpointIds = [];

				if (newId === undefined) {
					this.meterSelection = MeterSelection.NONE;
				}
			},
		},
	},
	methods: {
		meterTitle,
		meterSelectionChanged(selection: MeterSelection, values: { meter?: string }) {
			if (selection === MeterSelection.GRID) {
				values.meter = this.gridMeter?.name;
			} else if (selection === MeterSelection.NONE) {
				delete values.meter;
			} else if (values.meter === this.gridMeter?.name) {
				delete values.meter;
			}
		},
		handleConfigurationLoaded(values: DeviceValues) {
			if (!values["meter"]) {
				this.meterSelection = MeterSelection.NONE;
			} else if (this.gridMeter && values["meter"] === this.gridMeter.name) {
				this.meterSelection = MeterSelection.GRID;
			} else {
				this.meterSelection = MeterSelection.DEDICATED;
			}
			if (this.circuitName) {
				this.selectedLoadpointIds = this.initialAssignedLoadpoints();
			}
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
				this.meterSelection = MeterSelection.NONE;
				delete values["meter"];
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
				this.meterSelection = MeterSelection.DEDICATED;
				values.meter = result.name;
			} else if (result.action === "removed") {
				this.meterSelection = MeterSelection.NONE;
				delete values.meter;
			}
		},
		initialAssignedLoadpoints() {
			return this.availableLoadpoints.map((l) => l.id) as number[];
		},
		formatLoadpoints(loadpoints: ConfigLoadpoint[]) {
			const labels = loadpoints.map(({ title, name }) =>
				name ? `${title} (${name})` : title
			);

			return new Intl.ListFormat(this.$i18n?.locale).format(labels);
		},
		async patchAssignedLoadpoints(circuitDeleted?: boolean) {
			const initial = this.initialAssignedLoadpoints();
			const current = circuitDeleted ? [] : this.selectedLoadpointIds;

			const addedLoadpoints = current.filter((id) => !initial.includes(id));
			const removedLoadpoints = initial.filter((id) => !current.includes(id));

			await Promise.all([
				...addedLoadpoints.map((id) => this.patchLoadpoint(id, this.circuitName)),
				...removedLoadpoints.map((id) => this.patchLoadpoint(id)),
			]);
		},
		async patchLoadpoint(loadpointId: number, circuitName?: string) {
			await api.patch(
				`config/loadpoints/${loadpointId}`,
				{
					circuit: circuitName ?? null,
				},
				{
					headers: {
						"Content-Type": "application/merge-patch+json",
					},
				}
			);
		},
		async handleAdded(circuitName: string) {
			await this.patchAssignedLoadpoints();
			this.$emit("changed", circuitName);
		},
		async handleUpdated() {
			await this.patchAssignedLoadpoints();
			this.$emit("changed");
		},
		async handleRemoved() {
			await this.patchAssignedLoadpoints(true);
			this.$emit("changed");
		},
	},
});
</script>
