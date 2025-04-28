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
import { getSessionInfo, setSessionInfo } from "./session";
import type { CURRENCY, SelectOption } from "@/types/evcc";

export default defineComponent({
	name: "LoadpointSessionInfo",
	components: {
		LabelAndValue,
		CustomSelect,
	},
	mixins: [formatter],
	props: {
		id: { type: Number, default: 0 },
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
			selectedKey: getSessionInfo(this.id),
		};
	},
	computed: {
		options() {
			const result = [
				{
					key: "remaining",
					value: this.fmtDuration(this.chargeRemainingDurationInterpolated),
					available: this.chargeRemainingDurationInterpolated > 0,
				},
				{
					key: "finished",
					value: this.fmtHourMinute(this.finishTime),
					available: this.chargeRemainingDurationInterpolated > 0,
				},
				{
					key: "duration",
					value: this.fmtDuration(this.chargeDurationInterpolated),
				},
				{
					key: "solar",
					value: this.solarFormatted,
				},
				{
					key: "avgPrice",
					value: this.fmtAvgPrice(this.sessionPricePerKWh),
					valueSm: this.fmtAvgPriceShort(this.sessionPricePerKWh),
					available: this.tariffGrid !== undefined,
				},
				{
					key: "price",
					value: this.priceFormatted,
					available: this.tariffGrid !== undefined,
				},
				{
					key: "co2",
					value: this.fmtCo2Medium(this.sessionCo2PerKWh),
					valueSm: this.fmtCo2Short(this.sessionCo2PerKWh),
					available: this.tariffCo2 !== undefined,
				},
			];
			// only show options that are available
			return result.filter(({ available }) => available === undefined || available);
		},
		optionKeys() {
			return this.options.map((option) => option.key);
		},
		selectOptions(): SelectOption<string>[] {
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
			return this.$t(`main.loadpoint.${this.selectedOption.key}`);
		},
		value() {
			return this.selectedOption.value;
		},
		valueSm() {
			return this.selectedOption.valueSm;
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
			const index = this.optionKeys.indexOf(this.selectedKey);
			this.selectedKey = this.optionKeys[index + 1] || this.optionKeys[0];
			this.presist();
		},
		selectOption(value: string) {
			this.selectedKey = value;
			this.presist();
		},
		presist() {
			setSessionInfo(this.id, this.selectedKey);
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
