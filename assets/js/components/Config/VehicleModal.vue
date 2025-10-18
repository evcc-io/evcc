<template>
	<DeviceModalBase
		:id="id"
		modal-id="vehicleModal"
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
	>
		<template #collapsible-more="{ values }">
			<h6 class="mt-3">{{ $t("config.vehicle.chargingSettings") }}</h6>
			<FormRow
				id="vehicleParamMode"
				:label="$t('config.vehicle.defaultMode')"
				:help="$t('config.vehicle.defaultModeHelp')"
			>
				<PropertyField
					id="vehicleParamMode"
					v-model="values.mode"
					type="Choice"
					class="w-100"
					:choice="[
						{ key: 'off', name: $t('main.mode.off') },
						{ key: 'pv', name: $t('main.mode.pv') },
						{ key: 'minpv', name: $t('main.mode.minpv') },
						{ key: 'now', name: $t('main.mode.now') },
					]"
				/>
			</FormRow>
			<FormRow
				id="vehicleParamPhases"
				:label="$t('config.vehicle.maximumPhases')"
				:help="$t('config.vehicle.maximumPhasesHelp')"
			>
				<SelectGroup
					id="vehicleParamPhases"
					v-model="values.phases"
					class="w-100"
					:options="[
						{ name: '1-phase', value: '1' },
						{ name: '2-phases', value: '2' },
						{ name: '3-phases', value: '' },
					]"
					:aria-label="$t('config.vehicle.maximumPhases')"
					equal-width
					transparent
				/>
			</FormRow>
			<div class="row mb-3">
				<FormRow
					id="vehicleParamMinCurrent"
					:label="$t('config.vehicle.minimumCurrent')"
					class="col-sm-6 mb-sm-0"
					:help="
						values.minCurrent && values.minCurrent < 6
							? $t('config.vehicle.minimumCurrentHelp')
							: ''
					"
				>
					<PropertyField
						id="vehicleParamMinCurrent"
						v-model="values.minCurrent"
						type="Float"
						unit="A"
						size="w-25 w-min-200"
						class="me-2"
					/>
				</FormRow>
				<FormRow
					id="vehicleParamMaxCurrent"
					:label="$t('config.vehicle.maximumCurrent')"
					class="col-sm-6 mb-sm-0"
					:help="
						values.minCurrent &&
						values.maxCurrent &&
						values.maxCurrent < values.minCurrent
							? $t('config.vehicle.maximumCurrentHelp')
							: ''
					"
				>
					<PropertyField
						id="vehicleParamMaxCurrent"
						v-model="values.maxCurrent"
						type="Float"
						unit="A"
						size="w-25 w-min-200"
						class="me-2"
					/>
				</FormRow>
			</div>

			<FormRow
				id="vehicleParamPriority"
				:label="$t('config.vehicle.priority')"
				:help="$t('config.vehicle.priorityHelp')"
			>
				<PropertyField
					id="vehicleParamPriority"
					v-model="values.priority"
					type="Choice"
					size="w-100"
					class="me-2"
					:choice="priorityOptions"
					required
				/>
			</FormRow>

			<FormRow
				id="vehicleParamIdentifiers"
				:label="$t('config.vehicle.identifiers')"
				:help="$t('config.vehicle.identifiersHelp')"
			>
				<PropertyField
					id="vehicleParamIdentifiers"
					v-model="values.identifiers"
					type="List"
					property="identifiers"
					size="w-100"
					class="me-2"
				/>
			</FormRow>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import SelectGroup from "../Helper/SelectGroup.vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import { ConfigType } from "@/types/evcc";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import type { Product, ApiData, DeviceValues, TemplateParam } from "./DeviceModal";
import defaultVehicleYaml from "./defaultYaml/vehicle.yaml?raw";

const initialValues = {
	type: ConfigType.Template,
	icon: "car",
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};
const CUSTOM_FIELDS = ["minCurrent", "maxCurrent", "priority", "identifiers", "phases", "mode"];

export default defineComponent({
	name: "VehicleModal",
	components: {
		FormRow,
		PropertyField,
		SelectGroup,
		DeviceModalBase,
	},
	props: {
		id: Number,
		isSponsor: Boolean,
	},
	emits: ["vehicle-changed"],
	data() {
		return {
			initialValues,
		};
	},
	computed: {
		isNew(): boolean {
			return this.id === undefined;
		},
		priorityOptions() {
			const result: { key: number | undefined; name: string }[] = Array.from(
				{ length: 11 },
				(_, i) => ({ key: i, name: `${i}` })
			);
			result[0]!.name = "0 (default)";
			result[0]!.key = undefined;
			result[10]!.name = "10 (highest)";
			return result;
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
			const filtered = params
				.filter((p) => !CUSTOM_FIELDS.includes(p.Name))
				.map((p) => {
					if (p.Name === "title" || p.Name === "icon") {
						p.Required = true;
						p.Advanced = false;
					}
					return p;
				});

			filtered.sort((a, b) => (a.Required ? -1 : 1) - (b.Required ? -1 : 1));
			const order: Record<string, number> = { title: -2, icon: -1 };
			filtered.sort((a, b) => (order[a.Name] || 0) - (order[b.Name] || 0));

			return filtered;
		},
		transformApiData(data: ApiData, values: DeviceValues): ApiData {
			if (values.type === ConfigType.Custom) {
				delete data.icon;
				delete data.title;
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
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
</style>
