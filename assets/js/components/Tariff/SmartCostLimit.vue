<template>
	<SmartTariffBase
		v-bind="labels"
		:current-limit="currentLimit"
		:last-limit="lastLimit"
		:is-co2="isCo2"
		:currency="currency"
		:apply-all="multipleLoadpoints && isLoadpoint"
		:possible="possible"
		:tariff="tariff"
		:form-id="formId"
		:is-slot-active="isSlotActive"
		limit-direction="below"
		:options-start-at-zero="isCo2"
		@save-limit="saveLimit"
		@delete-limit="deleteLimit"
		@apply-to-all="applyToAll"
	/>
</template>

<script lang="ts">
import SmartTariffBase from "./SmartTariffBase.vue";
import { defineComponent, type PropType } from "vue";
import api from "@/api";
import { setLoadpointLastSmartCostLimit } from "@/uiLoadpoints";
import settings from "@/settings";
import { type CURRENCY, SMART_COST_TYPE } from "@/types/evcc";
import { type ForecastSlot } from "../Forecast/types";

export default defineComponent({
	name: "SmartCostLimit",
	components: { SmartTariffBase },
	props: {
		currentLimit: {
			type: [Number, null] as PropType<number | null>,
			required: true,
		},
		smartCostType: String as PropType<SMART_COST_TYPE>,
		currency: String as PropType<CURRENCY>,
		multipleLoadpoints: Boolean,
		isLoadpoint: Boolean,
		loadpointId: String,
		possible: Boolean,
		lastLimit: Number,
		tariff: Array as PropType<ForecastSlot[]>,
	},
	computed: {
		isCo2(): boolean {
			return this.smartCostType === SMART_COST_TYPE.CO2;
		},
		formId(): string {
			return `smartCostLimit-${this.loadpointId || "battery"}`;
		},
		labels() {
			const t = (key: string) => this.$t(`smartCost.${key}`);
			const co2 = this.isCo2;
			const lp = this.isLoadpoint;
			return {
				title: lp ? (co2 ? t("cleanTitle") : t("cheapTitle")) : "",
				description: lp ? t("loadpointDescription") : t("batteryDescription"),
				limitLabel: co2 ? t("co2Limit") : t("priceLimit"),
				currentPriceLabel: co2 ? t("co2Label") : t("priceLabel"),
				resetWarningText: t("resetWarning"),
				activeHoursLabel: t("activeHoursLabel"),
			};
		},
	},
	methods: {
		isSlotActive(value: number | undefined): boolean {
			if (value === undefined || this.currentLimit === null) {
				return false;
			}
			// Smart cost: charge when costs are below or equal to limit
			return value <= this.currentLimit;
		},
		async saveLimit(limit: number) {
			// save last selected value to be suggest again when reactivating limit
			this.saveLastLimit(limit);

			const url = this.isLoadpoint
				? `loadpoints/${this.loadpointId}/smartcostlimit`
				: "batterygridchargelimit";

			await api.post(`${url}/${encodeURIComponent(limit)}`);
		},
		saveLastLimit(limit: number) {
			if (this.isLoadpoint) {
				setLoadpointLastSmartCostLimit(this.loadpointId!, limit);
			} else {
				settings.lastBatterySmartCostLimit = limit;
			}
		},
		async deleteLimit() {
			// save last selected value to be suggest again when reactivating limit
			this.saveLastLimit(this.currentLimit || 0);

			const url = this.isLoadpoint
				? `loadpoints/${this.loadpointId}/smartcostlimit`
				: "batterygridchargelimit";

			await api.delete(url);
		},
		async applyToAll(selectedLimit: number | null) {
			if (selectedLimit === null) {
				await api.delete("smartcostlimit");
			} else {
				await api.post(`smartcostlimit/${encodeURIComponent(selectedLimit)}`);
			}
		},
	},
});
</script>
