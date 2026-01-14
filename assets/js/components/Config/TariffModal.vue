<template>
	<DeviceModalBase
		:id="id"
		modal-id="tariffModal"
		device-type="tariff"
		:modal-title="modalTitle"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:show-main-content="!!tariffType"
		:on-template-change="handleTemplateChange"
		@added="handleAdded"
		@updated="$emit('updated')"
		@removed="handleRemoved"
		@close="handleClose"
	>
		<template #pre-content>
			<div v-if="!tariffType" class="d-flex flex-column gap-4">
				<NewDeviceButton
					v-for="t in typeChoices"
					:key="t"
					:title="$t(`config.tariff.option.${t}`)"
					class="addButton"
					@click="selectType(t)"
				/>
			</div>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import NewDeviceButton from "./NewDeviceButton.vue";
import { ConfigType, type TariffType } from "@/types/evcc";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import type { Product, DeviceValues } from "./DeviceModal";
import tariffPriceYaml from "./defaultYaml/tariffPrice.yaml?raw";
import tariffCo2Yaml from "./defaultYaml/tariffCo2.yaml?raw";
import tariffSolarYaml from "./defaultYaml/tariffSolar.yaml?raw";

const initialValues = {
	type: ConfigType.Template,
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};

export default defineComponent({
	name: "TariffModal",
	components: {
		DeviceModalBase,
		NewDeviceButton,
	},
	props: {
		id: Number,
		type: { type: String as PropType<TariffType>, default: null },
		typeChoices: { type: Array as () => TariffType[], default: () => [] },
	},
	emits: ["added", "updated", "removed", "close"],
	data() {
		return {
			initialValues,
			selectedType: null as TariffType | null,
		};
	},
	computed: {
		isNew(): boolean {
			return this.id === undefined;
		},
		tariffType(): TariffType | null {
			return this.type || this.selectedType;
		},
		modalTitle(): string {
			if (this.isNew) {
				if (this.tariffType) {
					return this.$t(`config.tariff.${this.tariffType}.titleAdd`);
				} else {
					return this.$t("config.tariff.titleChoice");
				}
			}
			return this.$t(`config.tariff.${this.tariffType}.titleEdit`);
		},
	},
	methods: {
		provideTemplateOptions(products: Product[]): TemplateGroup[] {
			// Use different custom option text for tariffs vs forecasts
			const isForecast = ["co2", "planner", "solar"].includes(this.tariffType || "");
			const customLabel = isForecast
				? this.$t("config.tariff.customForecast")
				: this.$t("config.tariff.customTariff");

			// Separate demo/generic templates from real services
			const genericTemplates = [
				"demo-co2-forecast",
				"demo-dynamic-grid",
				"demo-solar-forecast",
				"energy-charts-api",
			];

			// Filter products by group upfront
			const filterByGroup = (group: string, onlyGeneric: boolean = false) =>
				products.filter((p: Product) => {
					const isGeneric = genericTemplates.includes(p.template);
					return p.group === group && (onlyGeneric ? isGeneric : !isGeneric);
				});

			const priceProducts = filterByGroup("price");
			const co2Products = filterByGroup("co2");
			const solarProducts = filterByGroup("solar");
			const priceGeneric = filterByGroup("price", true);
			const co2Generic = filterByGroup("co2", true);
			const solarGeneric = filterByGroup("solar", true);

			// Special handling for planner: show price + co2 services
			if (this.tariffType === "planner") {
				return [
					{
						label: "generic",
						options: [
							...priceGeneric,
							...co2Generic,
							customTemplateOption(customLabel),
						],
					},
					{
						label: "services",
						options: priceProducts,
					},
					{
						label: "co2Services",
						options: co2Products,
					},
				];
			}

			// Map tariff types to product groups
			const groupMap: Record<string, { service: Product[]; generic: Product[] }> = {
				grid: { service: priceProducts, generic: priceGeneric },
				feedin: { service: priceProducts, generic: priceGeneric },
				co2: { service: co2Products, generic: co2Generic },
				solar: { service: solarProducts, generic: solarGeneric },
			};
			const mapped = (this.tariffType && groupMap[this.tariffType]) || {
				service: [],
				generic: [],
			};

			return [
				{
					label: "generic",
					options: [...mapped.generic, customTemplateOption(customLabel)],
				},
				{
					label: "services",
					options: mapped.service,
				},
			];
		},
		handleTemplateChange(e: Event, values: DeviceValues) {
			const value = (e.target as HTMLSelectElement).value;
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				// Select appropriate YAML template based on tariff type
				if (
					this.tariffType === "grid" ||
					this.tariffType === "feedin" ||
					this.tariffType === "planner"
				) {
					values.yaml = tariffPriceYaml;
				} else if (this.tariffType === "co2") {
					values.yaml = tariffCo2Yaml;
				} else if (this.tariffType === "solar") {
					values.yaml = tariffSolarYaml;
				}
			}
		},
		selectType(type: TariffType) {
			this.selectedType = type;
		},
		handleAdded(name: string) {
			this.$emit("added", this.tariffType, name);
		},
		handleRemoved() {
			this.$emit("removed", this.tariffType);
		},
		handleClose() {
			this.selectedType = null;
			this.$emit("close");
		},
	},
});
</script>
<style scoped>
.addButton {
	min-height: 6rem;
}
</style>
