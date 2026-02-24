<template>
	<SmartTariffBase
		v-bind="labels"
		:current-limit="currentLimit"
		:last-limit="lastLimit"
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
import { setLoadpointLastSmartFeedInPriorityLimit } from "@/uiLoadpoints";

export default defineComponent({
	name: "SmartFeedInPriority",
	components: { SmartTariffBase },
	props: {
		currentLimit: {
			type: [Number, null] as PropType<number | null>,
			required: true,
		},
		lastLimit: Number,
		currency: String as PropType<CURRENCY>,
		loadpointId: { type: String, required: true },
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
		async saveLimit(limit: number) {
			// save last selected value to be suggest again when reactivating limit
			this.saveLastLimit(limit);

			const url = `loadpoints/${this.loadpointId}/smartfeedinprioritylimit`;
			await api.post(`${url}/${encodeURIComponent(limit)}`);
		},
		saveLastLimit(limit: number) {
			setLoadpointLastSmartFeedInPriorityLimit(this.loadpointId, limit);
		},
		async deleteLimit() {
			// save last selected value to be suggest again when reactivating limit
			this.saveLastLimit(this.currentLimit || 0);

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
