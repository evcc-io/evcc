<template>
	<div class="sessionInfo">
		<LabelAndValue class="d-block" align="end">
			<template #label>
				<CustomSelect
					inline
					class="align-top"
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
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import { getLoadpointSessionInfo, setLoadpointSessionInfo } from "@/uiLoadpoints";
import type {
	CURRENCY,
	PeriodStats,
	SelectOption,
	SessionInfoKey,
	SessionInfoPeriod,
	SessionInfoPeriodValue,
} from "@/types/evcc";

interface Option {
	key: SessionInfoKey;
	name: string; // dropdown entry
	label: string; // card label, names the period outside the dropdown
	value: string;
	valueSm?: string;
	group?: string;
	available: boolean;
}

type OptionExtras = Partial<Pick<Option, "valueSm" | "available">>;

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
		sessionEnergy: { type: Number, default: 0 },
		last24h: Object as PropType<PeriodStats>,
		last7d: Object as PropType<PeriodStats>,
		tariffCo2: Number,
		tariffGrid: Number,
		experimental: Boolean,
		chargerFeatureIntegratedDevice: Boolean,
	},
	data() {
		return {
			selectedKey: this.id ? getLoadpointSessionInfo(this.id) : undefined,
		};
	},
	computed: {
		sessionOptions(): Option[] {
			const group = this.$t("main.loadpoint.session");
			const option = (
				key: SessionInfoKey,
				value: string,
				{ valueSm, available = true }: OptionExtras = {}
			): Option => {
				const name = this.$t(`main.loadpoint.${key}`);
				return { key, name, label: name, value, valueSm, group, available };
			};
			const remaining = this.chargeRemainingDurationInterpolated > 0;
			const priced = this.tariffGrid !== undefined;
			const co2 = this.tariffCo2 !== undefined;
			return [
				option("remaining", this.fmtDuration(this.chargeRemainingDurationInterpolated), {
					available: remaining,
				}),
				option("finished", this.fmtHourMinute(this.finishTime), { available: remaining }),
				option("duration", this.fmtDuration(this.chargeDurationInterpolated)),
				option("solar", this.fmtPercentage(this.sessionSolarPercentage, 1)),
				option("avgPrice", this.fmtAvgPrice(this.sessionPricePerKWh), {
					valueSm: this.fmtAvgPriceShort(this.sessionPricePerKWh),
					available: priced,
				}),
				option("price", this.fmtMoneyWithSymbol(this.sessionPrice, this.currency), {
					available: priced,
				}),
				option("co2", this.fmtCo2Medium(this.sessionCo2PerKWh), {
					valueSm: this.fmtCo2Short(this.sessionCo2PerKWh),
					available: co2,
				}),
				option("emission", this.fmtEmission(this.sessionCo2PerKWh, this.sessionEnergy), {
					available: co2,
				}),
			];
		},
		periodOptions(): Option[] {
			return [
				...this.optionsForPeriod("last24h", this.last24h),
				...this.optionsForPeriod("last7d", this.last7d),
			];
		},
		options(): Option[] {
			const periods = this.experimental ? this.periodOptions : [];
			// integrated devices drop the session once periods replace it
			const session =
				this.chargerFeatureIntegratedDevice && periods.length ? [] : this.sessionOptions;
			return [...session, ...periods].filter((option) => option.available);
		},
		optionKeys(): SessionInfoKey[] {
			return this.options.map((option) => option.key);
		},
		selectOptions(): SelectOption<SessionInfoKey>[] {
			return this.options.map(({ key, name, group }) => ({ name, value: key, group }));
		},
		selectedOption(): Option | undefined {
			return (
				this.options.find((option) => option.key === this.selectedKey) || this.options[0]
			);
		},
		label() {
			return this.selectedOption?.label;
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
	},
	methods: {
		optionsForPeriod(prefix: SessionInfoPeriod, stats?: PeriodStats): Option[] {
			if (!stats) return [];
			const group = this.$t(`main.loadpoint.${prefix}Energy`);
			const short = this.$t(`main.loadpoint.${prefix}Short`);
			const option = (
				suffix: SessionInfoPeriodValue,
				labelKey: string,
				value: string,
				{ valueSm, available = true }: OptionExtras = {}
			): Option => {
				const name = this.$t(`main.loadpoint.${labelKey}`);
				return {
					key: `${prefix}${suffix}` as SessionInfoKey,
					name,
					label: `${name} (${short})`,
					value,
					valueSm,
					group,
					available,
				};
			};
			// device cards say charged for vehicles and consumed for heaters
			const energyKey = this.chargerFeatureIntegratedDevice ? "consumed" : "charged";
			const { energy, autarky, pricePerKWh, price } = stats;
			const priced = pricePerKWh != null;
			return [
				option("Energy", energyKey, this.fmtWh(energy, POWER_UNIT.AUTO)),
				option("Autarky", "autarky", this.fmtPercentage(autarky ?? 0, 1), {
					available: autarky != null,
				}),
				option("AvgPrice", "avgPrice", this.fmtAvgPrice(pricePerKWh ?? 0), {
					valueSm: this.fmtAvgPriceShort(pricePerKWh ?? 0),
					available: priced,
				}),
				option("Price", "price", this.fmtMoneyWithSymbol(price ?? 0, this.currency), {
					available: priced,
				}),
			];
		},
		fmtAvgPrice(value: number) {
			return this.fmtPricePerKWh(value, this.currency, false);
		},
		fmtAvgPriceShort(value: number) {
			return this.fmtPricePerKWh(value, this.currency, true);
		},
		fmtEmission(co2PerKWh: number, energyWh: number) {
			return this.fmtGrams((co2PerKWh * energyWh) / 1000);
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
	user-select: none;
	-webkit-user-select: none;
}
.sessionInfo .value {
	cursor: pointer;
}
</style>
