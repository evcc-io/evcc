<template>
	<GenericModal
		id="vehicleModal"
		:title="$t(`config.vehicle.${isNew ? 'titleAdd' : 'titleEdit'}`)"
		:size="modalSize"
		data-testid="vehicle-modal"
		@open="open"
		@closed="closed"
	>
		<form ref="form" class="container mx-0 px-0">
			<TemplateSelector
				ref="templateSelect"
				v-model="templateName"
				device-type="vehicle"
				:is-new="isNew"
				:product-name="productName"
				:groups="templateOptions"
				@change="templateChanged"
			/>
			<YamlEntry
				v-if="values.type === 'custom'"
				v-model="values.yaml"
				type="vehicle"
				:error-line="test.errorLine"
			/>
			<div v-else>
				<p v-if="loadingTemplate">{{ $t("config.general.templateLoading") }}</p>
				<Markdown v-if="description" :markdown="description" class="my-4" />
				<PropertyEntry
					v-for="param in normalParams"
					:id="`vehicleParam${param.Name}`"
					:key="param.Name"
					v-bind="param"
					v-model="values[param.Name]"
				/>

				<PropertyCollapsible>
					<template v-if="advancedParams.length" #advanced>
						<PropertyEntry
							v-for="param in advancedParams"
							:id="`vehicleParam${param.Name}`"
							:key="param.Name"
							v-bind="param"
							v-model="values[param.Name]"
						/>
					</template>
					<template #more>
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
				</PropertyCollapsible>
			</div>

			<DeviceModalActions
				v-if="showActions"
				:is-deletable="isDeletable"
				:test-state="test"
				:is-saving="saving"
				@save="isNew ? create() : update()"
				@remove="remove"
				@test="testManually"
			/>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import SelectGroup from "../Helper/SelectGroup.vue";
import PropertyEntry from "./PropertyEntry.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import GenericModal from "../Helper/GenericModal.vue";
import Markdown from "./Markdown.vue";
import TemplateSelector, { customTemplateOption } from "./DeviceModal/TemplateSelector.vue";
import DeviceModalActions from "./DeviceModal/Actions.vue";
import YamlEntry from "./DeviceModal/YamlEntry.vue";
import { initialTestState, performTest } from "./utils/test";
import { ConfigType } from "@/types/evcc";
import {
	handleError,
	type DeviceValues,
	type Template,
	type Product,
	applyDefaultsFromTemplate,
	createDeviceUtils,
} from "./DeviceModal";
import defaultYaml from "./defaultYaml/vehicle.yaml?raw";

const initialValues = { type: ConfigType.Template, icon: "car" };
const device = createDeviceUtils("vehicle");

function sleep(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

const CUSTOM_FIELDS = ["minCurrent", "maxCurrent", "priority", "identifiers", "phases", "mode"];

type VehicleDeviceValues = DeviceValues & {
	identifiers: string[] | undefined;
	minCurrent: number | undefined;
	maxCurrent: number | undefined;
	priority: number | undefined;
	phases: number | undefined;
	mode: string | undefined;
};

export default defineComponent({
	name: "VehicleModal",
	components: {
		FormRow,
		PropertyField,
		GenericModal,
		SelectGroup,
		PropertyCollapsible,
		PropertyEntry,
		Markdown,
		TemplateSelector,
		DeviceModalActions,
		YamlEntry,
	},
	props: {
		id: Number,
	},
	emits: ["vehicle-changed"],
	data() {
		return {
			isModalVisible: false,
			templates: [] as Template[],
			products: [] as Product[],
			templateName: null as string | null,
			template: null as Template | null,
			saving: false,
			loadingTemplate: false,
			values: { ...initialValues } as VehicleDeviceValues,
			test: initialTestState(),
		};
	},
	computed: {
		templateOptions() {
			return [
				{
					label: "primary",
					options: [
						...this.products.filter((p) => p.template === "offline"),
						customTemplateOption(this.$t("config.general.customOption")),
					],
				},
				{
					label: "online",
					options: this.products.filter((p) => !p.group),
				},
				{
					label: "scooter",
					options: this.products.filter((p) => p.group === "scooter"),
				},
				{
					label: "generic",
					options: this.products.filter(
						(p) => p.group === "generic" && p.template !== "offline"
					),
				},
			];
		},
		templateParams() {
			const params = (this.template?.Params || [])
				.filter((p) => !CUSTOM_FIELDS.includes(p.Name))
				.map((p) => {
					if (p.Name === "title" || p.Name === "icon") {
						p.Required = true;
						p.Advanced = false;
					}
					return p;
				});

			// non-optional fields first
			params.sort((a, b) => (a.Required ? -1 : 1) - (b.Required ? -1 : 1));
			// always start with title and icon field
			const order: Record<string, number> = { title: -2, icon: -1 };
			params.sort((a, b) => (order[a.Name] || 0) - (order[b.Name] || 0));

			return params;
		},
		normalParams() {
			return this.templateParams.filter(
				(p) =>
					!p.Advanced && !p.Deprecated && (p.Usages ? p.Usages.includes("vehicle") : true)
			);
		},
		advancedParams() {
			return this.templateParams.filter(
				(p) =>
					(p.Advanced || p.Deprecated) && (p.Usages ? p.Usages.includes("vehicle") : true)
			);
		},
		description() {
			return this.template?.Requirements?.Description;
		},
		productName() {
			return this.values.deviceProduct || this.templateName || "";
		},
		apiData() {
			const data: Record<string, any> = {
				...this.values,
			};
			if (this.values.type === ConfigType.Template) {
				data["template"] = this.templateName;
			}
			// remove icon if custom
			if (this.values.type === ConfigType.Custom) {
				delete data["icon"];
				delete data["title"];
			}
			// trim and remove empty lines
			if (Array.isArray(data["identifiers"])) {
				data["identifiers"] = data["identifiers"].map((i) => i.trim()).filter((i) => i);
			}
			return data;
		},
		isNew() {
			return this.id === undefined;
		},
		modalSize() {
			return this.values.type === ConfigType.Custom ? "xl" : undefined;
		},
		isDeletable() {
			return !this.isNew;
		},
		priorityOptions() {
			const result: { key: number | undefined; name: string }[] = Array.from(
				{ length: 11 },
				(_, i) => ({ key: i, name: `${i}` })
			);
			result[0].name = "0 (default)";
			result[0].key = undefined;
			result[10].name = "10 (highest)";
			return result;
		},
		showActions() {
			return this.templateName || this.values.type === ConfigType.Custom;
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				this.reset();
				this.templateName = "offline";
				this.test = initialTestState();
				this.loadProducts();
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
		},
		templateName() {
			this.loadTemplate();
		},
		values: {
			handler() {
				this.test = initialTestState();
			},
			deep: true,
		},
	},
	methods: {
		reset() {
			this.values = { ...initialValues } as VehicleDeviceValues;
			this.test = initialTestState();
		},
		async loadConfiguration() {
			try {
				const vehicle = await device.load(this.id as number);
				this.values = vehicle.config;
				// convert structure to flat list
				// TODO: adjust GET response to match POST/PUT formats
				this.values.type = vehicle.type;
				this.values.deviceProduct = vehicle.deviceProduct;
				applyDefaultsFromTemplate(this.template, this.values);
				this.templateName = this.values.template;
			} catch (e) {
				console.error(e);
			}
		},
		async loadProducts() {
			if (!this.isModalVisible) {
				return;
			}
			try {
				this.products = await device.loadProducts(this.$i18n?.locale);
			} catch (e) {
				console.error(e);
			}
		},
		async loadTemplate() {
			this.template = null;
			if (!this.templateName || this.templateName === ConfigType.Custom) return;
			this.loadingTemplate = true;
			try {
				this.template = await device.loadTemplate(this.templateName, this.$i18n?.locale);
				applyDefaultsFromTemplate(this.template, this.values);
			} catch (e) {
				console.error(e);
			}
			this.loadingTemplate = false;
		},
		async create() {
			// persist selected template product
			if (this.template && this.$refs["templateSelect"]) {
				this.values.deviceProduct = (this.$refs["templateSelect"] as any).getProductName();
			}
			if (this.test.isUnknown) {
				const success = await performTest(this.test, this.testVehicle, this.$refs["form"]);
				if (!success) return;
				await sleep(100);
			}
			this.saving = true;
			try {
				const { name } = await device.create(this.apiData);
				this.$emit("vehicle-changed", name);
				this.closed();
			} catch (e) {
				handleError(e, "create failed");
			}
			this.saving = false;
		},
		async testManually() {
			await performTest(this.test, this.testVehicle, this.$refs["form"]);
		},
		async testVehicle() {
			return device.test(this.id, this.apiData);
		},
		async update() {
			if (this.test.isUnknown) {
				const success = await performTest(this.test, this.testVehicle, this.$refs["form"]);
				if (!success) return;
				await sleep(250);
			}
			this.saving = true;
			try {
				await device.update(this.id as number, this.apiData);
				this.$emit("vehicle-changed");
				this.closed();
			} catch (e) {
				handleError(e, "update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await device.remove(this.id as number);
				this.$emit("vehicle-changed");
				this.closed();
			} catch (e) {
				handleError(e, "remove failed");
			}
		},
		open() {
			this.isModalVisible = true;
		},
		closed() {
			this.isModalVisible = false;
		},
		templateChanged(e: Event) {
			const value = (e.target as HTMLSelectElement).value;
			this.reset();
			if (value === ConfigType.Custom) {
				this.values.type = ConfigType.Custom;
				this.values.yaml = defaultYaml;
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
