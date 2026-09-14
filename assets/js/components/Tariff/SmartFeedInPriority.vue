<template>
	<SmartTariffBase
		v-bind="labels"
		:current-limit="currentLimit"
		:last-limit="lastLimit"
		:currency="currency"
		:apply-all="isLoadpoint && multipleLoadpoints"
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
import settings from "@/settings";
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
		loadpointId: String,
		isLoadpoint: Boolean,
		multipleLoadpoints: Boolean,
		possible: Boolean,
		tariff: Array,
	},
	computed: {
		formId(): string {
			return `smartFeedInPriority-${this.loadpointId || "battery"}`;
		},
		labels() {
			const t = (key: string) => this.$t(`smartFeedInPriority.${key}`);
			return {
				title: this.isLoadpoint ? t("title") : "",
				description: this.isLoadpoint ? t("description") : t("batteryDescription"),
				limitLabel: t("priceLimit"),
				activeHoursLabel: t("activeHoursLabel"),
				currentPriceLabel: t("priceLabel"),
				resetWarningKey: "smartFeedInPriority.resetWarning",
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
		async saveLimit(limit: number, active: boolean) {
			// save last selected value to be suggest again when reactivating limit
			this.saveLastLimit(limit);

			if (!active) return;

			const url = this.isLoadpoint
				? `loadpoints/${this.loadpointId}/smartfeedinprioritylimit`
				: "batterygriddischargelimit";

			await api.post(`${url}/${encodeURIComponent(limit)}`);
		},
		saveLastLimit(limit: number) {
			if (this.isLoadpoint) {
				setLoadpointLastSmartFeedInPriorityLimit(this.loadpointId!, limit);
			} else {
				settings.lastBatteryGridDischargeLimit = limit;
			}
		},
		async deleteLimit() {
			// save last selected value to be suggest again when reactivating limit
			this.saveLastLimit(this.currentLimit || 0);

			const url = this.isLoadpoint
				? `loadpoints/${this.loadpointId}/smartfeedinprioritylimit`
				: "batterygriddischargelimit";

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
