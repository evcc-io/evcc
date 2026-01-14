<template>
	<DeviceCard
		:title="cardTitle"
		:name="tariff.name"
		:editable="!!tariff.id"
		:error="hasError"
		:data-testid="tariffType"
		@edit="$emit('edit', tariffType, tariff.id)"
	>
		<template #icon>
			<component :is="iconComponent" />
		</template>
		<template #tags>
			<DeviceTags :tags="tags" />
		</template>
	</DeviceCard>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/invoice";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/clock";
import "@h2d2/shopicons/es/regular/sun";
import { defineComponent, type PropType } from "vue";
import DeviceCard from "./DeviceCard.vue";
import DeviceTags from "./DeviceTags.vue";
import type { TariffType } from "@/types/evcc";

type ConfigTariff = {
	id: number;
	name: string;
	config?: {
		template?: string;
	};
};

export default defineComponent({
	name: "TariffCard",
	components: {
		DeviceCard,
		DeviceTags,
	},
	props: {
		tariff: { type: Object as PropType<ConfigTariff>, required: true },
		tariffType: { type: String as PropType<TariffType>, required: true },
		hasError: { type: Boolean, default: false },
		title: String,
		tags: { type: Object, default: () => ({}) },
	},
	emits: ["edit"],
	computed: {
		cardTitle(): string {
			if (this.title) {
				return this.title;
			}
			if (this.tariff.config?.template) {
				return this.tariff.config.template;
			}
			return this.fallbackTitle;
		},
		fallbackTitle(): string {
			return this.$t(`config.tariff.type.${this.tariffType}`);
		},
		iconComponent(): string {
			const iconMap: Record<TariffType, string> = {
				grid: "shopicon-regular-invoice",
				feedin: "shopicon-regular-invoice",
				co2: "shopicon-regular-eco1",
				planner: "shopicon-regular-clock",
				solar: "shopicon-regular-sun",
			};
			return iconMap[this.tariffType];
		},
	},
});
</script>
