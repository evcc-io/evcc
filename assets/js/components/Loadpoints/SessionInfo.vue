<template>
	<div class="sessionInfo">
		<LabelAndValue class="d-block" align="end">
			<template #label>
				<CustomSelect
					:selected="selectedKey"
					:options="selectOptions"
					data-testid="sessionInfoSelect"
					@change="selectOption($event.target.value)"
				>
					<div
						class="text-decoration-underline text-truncate-xs-only"
						data-testid="sessionInfoLabel"
					>
						{{ label }}
					</div>
				</CustomSelect>
			</template>
			<template #value>
				<div class="value" data-testid="sessionInfoValue" @click="nextSessionInfo">
					<div :class="{ 'd-none d-sm-block': showSm }">{{ value }}</div>
					<div v-if="showSm" class="d-block d-sm-none">{{ valueSm }}</div>
				</div>
			</template>
		</LabelAndValue>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import LabelAndValue from "../Helper/LabelAndValue.vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import formatter from "@/mixins/formatter";
import { getLoadpointSessionInfo, setLoadpointSessionInfo } from "@/uiLoadpoints";
import type { CURRENCY, SelectOption, SessionInfoKey } from "@/types/evcc";

export default defineComponent({
	name: "LoadpointSessionInfo",
	components: {
		LabelAndValue,
		CustomSelect,
	},
	mixins: [formatter],
	props: {
		id: String,
		sessionCo2PerKWh: { type: Number, default: 0 },
		sessionPricePerKWh: { type: Number, default: 0 },
		sessionPrice: { type: Number, default: 0 },
		currency: String as PropType<CURRENCY>,
		sessionSolarPercentage: { type: Number, default: 0 },
		chargeRemainingDurationInterpolated: { type: Number, default: 0 },
		chargeDurationInterpolated: Number,
		tariffCo2: Number,
		tariffGrid: Number,
	},
	data() {
		return {
			selectedKey: this.id ? getLoadpointSessionInfo(this.id) : undefined,
		};
	},
	computed: {
		options(): Array<{
			key: SessionInfoKey;
			value: string;
			valueSm?: string;
			available?: boolean;
		}> {
			const result = [
				{
					key: "remaining" as const,
					value: this.fmtDuration(this.chargeRemainingDurationInterpolated),
					available: this.chargeRemainingDurationInterpolated > 0,
				},
				{
					key: "finished" as const,
					value: this.fmtHourMinute(this.finishTime),
					available: this.chargeRemainingDurationInterpolated > 0,
				},
				{
					key: "duration" as const,
					value: this.fmtDuration(this.chargeDurationInterpolated),
				},
				{
					key: "solar" as const,
					value: this.solarFormatted,
				},
				{
					key: "avgPrice" as const,
					value: this.fmtAvgPrice(this.sessionPricePerKWh),
					valueSm: this.fmtAvgPriceShort(this.sessionPricePerKWh),
					available: this.tariffGrid !== undefined,
				},
				{
					key: "price" as const,
					value: this.priceFormatted,
					available: this.tariffGrid !== undefined,
				},
				{
					key: "co2" as const,
					value: this.fmtCo2Medium(this.sessionCo2PerKWh),
					valueSm: this.fmtCo2Short(this.sessionCo2PerKWh),
					available: this.tariffCo2 !== undefined,
				},
			];
			// only show options that are available
			return result.filter(({ available }) => available === undefined || available);
		},
		optionKeys(): SessionInfoKey[] {
			return this.options.map((option) => option.key);
		},
		selectOptions(): SelectOption<SessionInfoKey>[] {
			return this.optionKeys.map((key) => ({
				name: this.$t(`main.loadpoint.${key}`),
				value: key,
			}));
		},
		selectedOption() {
			return (
				this.options.find((option) => option.key === this.selectedKey) || this.options[0]
			);
		},
		label() {
			return this.$t(`main.loadpoint.${this.selectedOption?.key || ""}`);
		},
		value() {
			return this.selectedOption?.value;
		},
		valueSm() {
			return this.selectedOption?.valueSm;
		},
		showSm() {
			return this.valueSm !== undefined;
		},
		finishTime() {
			const remainingSeconds = this.chargeRemainingDurationInterpolated;
			const now = new Date();
			if (remainingSeconds > 0) {
				return new Date(now.getTime() + remainingSeconds * 1000);
			}
			return now;
		},
		solarFormatted() {
			return this.fmtPercentage(this.sessionSolarPercentage, 1);
		},
		priceFormatted() {
			return `${this.fmtMoney(this.sessionPrice, this.currency)} ${this.fmtCurrencySymbol(
				this.currency
			)}`;
		},
	},
	methods: {
		fmtAvgPrice(value: number) {
			return this.fmtPricePerKWh(value, this.currency, false);
		},
		fmtAvgPriceShort(value: number) {
			return this.fmtPricePerKWh(value, this.currency, true);
		},
		nextSessionInfo() {
			const index = this.selectedKey ? this.optionKeys.indexOf(this.selectedKey) : -1;
			this.selectedKey = this.optionKeys[index + 1] || this.optionKeys[0];
			this.presist();
		},
		selectOption(value: SessionInfoKey) {
			this.selectedKey = value;
			this.presist();
		},
		presist() {
			if (this.selectedKey && this.id) {
				setLoadpointSessionInfo(this.id, this.selectedKey);
			}
		},
	},
});
</script>

<style scoped>
.sessionInfo * {
	cursor: pointer;
	user-select: none;
	-webkit-user-select: none;
}
</style>
