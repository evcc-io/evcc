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
					<span class="text-decoration-underline" data-testid="sessionInfoLabel">
						{{ label }}
					</span>
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

<script>
import LabelAndValue from "./LabelAndValue.vue";
import formatter from "../mixins/formatter";
import CustomSelect from "./CustomSelect.vue";
import { getSessionInfo, setSessionInfo } from "../sessionInfo";

export default {
	name: "LoadpointSessionInfo",
	components: {
		LabelAndValue,
		CustomSelect,
	},
	mixins: [formatter],
	props: {
		id: Number,
		sessionEnergy: Number,
		sessionCo2PerKWh: Number,
		sessionPricePerKWh: Number,
		sessionPrice: Number,
		currency: String,
		sessionSolarPercentage: Number,
		chargeRemainingDurationInterpolated: Number,
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
		options: function () {
			const result = [
				{
					key: "remaining",
					value: this.fmtDuration(this.chargeRemainingDurationInterpolated),
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
		selectOptions() {
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
		solarFormatted() {
			return `${this.fmtNumber(this.sessionSolarPercentage, 1)}%`;
		},
		priceFormatted() {
			return `${this.fmtMoney(this.sessionPrice, this.currency)} ${this.fmtCurrencySymbol(
				this.currency
			)}`;
		},
	},
	methods: {
		fmtAvgPrice(value) {
			return this.fmtPricePerKWh(value, this.currency, false);
		},
		fmtAvgPriceShort(value) {
			return this.fmtPricePerKWh(value, this.currency, true);
		},
		nextSessionInfo() {
			const index = this.optionKeys.indexOf(this.selectedKey);
			this.selectedKey = this.optionKeys[index + 1] || this.optionKeys[0];
			this.presist();
		},
		selectOption(value) {
			this.selectedKey = value;
			this.presist();
		},
		presist() {
			setSessionInfo(this.id, this.selectedKey);
		},
	},
};
</script>

<style scoped>
.sessionInfo * {
	cursor: pointer;
	user-select: none;
	-webkit-user-select: none;
}
</style>
