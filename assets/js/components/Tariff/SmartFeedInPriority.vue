<template>
	<SmartTariffBase
		v-bind="labels"
		:current-limit="currentLimit"
		:currency="currency"
		:apply-all="multipleLoadpoints"
		:possible="possible"
		:tariff="tariff"
		:form-id="formId"
		:is-slot-active="isSlotActive"
		options-extra-high
		options-start-at-zero
		limit-direction="above"
		highlight-color="text-warning"
		@save-limit="saveLimit"
		@delete-limit="deleteLimit"
		@apply-to-all="applyToAll"
	/>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import SmartTariffBase from "./SmartTariffBase.vue";
import api from "@/api";
import { type CURRENCY } from "@/types/evcc";

export default defineComponent({
	name: "SmartFeedInPriority",
	components: { SmartTariffBase },
	props: {
		currentLimit: {
			type: [Number, null] as PropType<number | null>,
			required: true,
		},
		currency: String as PropType<CURRENCY>,
		loadpointId: { type: Number, required: true },
		multipleLoadpoints: Boolean,
		possible: Boolean,
		tariff: Array,
	},
	computed: {
		formId(): string {
			return `smartFeedInPriority-${this.loadpointId}`;
		},
		labels() {
			const t = (key: string) => this.$t(`smartFeedInPriority.${key}`);
			return {
				title: t("title"),
				description: t("description"),
				limitLabel: t("priceLimit"),
				activeHoursLabel: t("activeHoursLabel"),
				currentPriceLabel: t("priceLabel"),
				resetWarningText: t("resetWarning"),
			};
		},
	},
	methods: {
		isSlotActive(value: number | undefined): boolean {
			if (value === undefined || this.currentLimit === null) {
				return false;
			}
			// Smart feed-in priority: pause when rates are above or equal to limit
			return value >= this.currentLimit;
		},
		async saveLimit(limit: string) {
			const url = `loadpoints/${this.loadpointId}/smartfeedinprioritylimit`;

			if (limit === "null") {
				await api.delete(url);
			} else {
				await api.post(`${url}/${encodeURIComponent(limit)}`);
			}
		},
		async deleteLimit() {
			const url = `loadpoints/${this.loadpointId}/smartfeedinprioritylimit`;
			await api.delete(url);
		},
		async applyToAll(selectedLimit: number | null) {
			if (selectedLimit === null) {
				await api.delete("smartfeedinprioritylimit");
			} else {
				await api.post(`smartfeedinprioritylimit/${encodeURIComponent(selectedLimit)}`);
			}
		},
	},
});
</script>
