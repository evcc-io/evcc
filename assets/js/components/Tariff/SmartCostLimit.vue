<template>
	<SmartTariffBase
		v-bind="labels"
		:current-limit="effectiveLimit"
		:last-limit="lastLimit"
		:is-co2="isCo2"
		:currency="currency"
		:apply-all="multipleLoadpoints && isLoadpoint && currentLimitPercent === null"
		:possible="possible"
		:tariff="tariff"
		:form-id="formId"
		:is-slot-active="isSlotActive"
		:relative-limit-supported="relativeLimitSupported"
		:relative-limit-label="relativeLimitLabel"
		:relative-limit-percent="currentLimitPercent"
		:relative-limit-value="relativeLimitValue"
		limit-direction="below"
		:options-start-at-zero="isCo2"
		@save-limit="saveLimit"
		@delete-limit="deleteLimit"
		@save-relative-limit="saveRelativeLimit"
		@delete-relative-limit="deleteRelativeLimit"
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
		currentLimitPercent: {
			type: [Number, null] as PropType<number | null>,
			default: null,
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
		relativeLimitSupported(): boolean {
			return this.isLoadpoint && !this.isCo2;
		},
		relativeLimitLabel(): string {
			return this.$t("smartCost.relativeLimitLabel");
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
		averageTariffValue(): number | null {
			if (!this.tariff?.length) {
				return null;
			}

			let sum = 0;
			let total = 0;
			this.tariff.forEach((slot) => {
				const start = new Date(slot.start);
				const end = new Date(slot.end);
				const duration = end.getTime() - start.getTime();
				if (!duration || Number.isNaN(duration)) {
					return;
				}
				sum += slot.value * duration;
				total += duration;
			});

			return total ? sum / total : null;
		},
		relativeLimitValue(): number | null {
			if (this.currentLimitPercent === null) {
				return null;
			}
			if (this.averageTariffValue === null) {
				return null;
			}
			return (this.averageTariffValue * this.currentLimitPercent) / 100;
		},
		effectiveLimit(): number | null {
			if (this.currentLimitPercent !== null) {
				return this.relativeLimitValue;
			}
			return this.currentLimit;
		},
	},
	methods: {
		isSlotActive(value: number | undefined): boolean {
			if (value === undefined || this.effectiveLimit === null) {
				return false;
			}
			// Smart cost: charge when costs are below or equal to limit
			return value <= this.effectiveLimit;
		},
		async saveLimit(limit: number) {
			if (this.currentLimitPercent !== null && this.relativeLimitSupported) {
				await this.deleteRelativeLimit();
			}
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
			if (this.currentLimitPercent !== null && this.relativeLimitSupported) {
				await this.deleteRelativeLimit();
				return;
			}
			// save last selected value to be suggest again when reactivating limit
			this.saveLastLimit(this.currentLimit || 0);

			const url = this.isLoadpoint
				? `loadpoints/${this.loadpointId}/smartcostlimit`
				: "batterygridchargelimit";

			await api.delete(url);
		},
		async saveRelativeLimit(percent: number) {
			if (!this.relativeLimitSupported) {
				return;
			}

			const url = `loadpoints/${this.loadpointId}/smartcostlimit/relative`;
			await api.post(`${url}/${encodeURIComponent(percent)}`);
		},
		async deleteRelativeLimit() {
			if (!this.relativeLimitSupported) {
				return;
			}

			const url = `loadpoints/${this.loadpointId}/smartcostlimit/relative`;
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
