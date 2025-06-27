<template>
	<div v-if="possible">
		<h6 v-if="title" class="mt-0">{{ title }}</h6>
		<p>
			{{ description }}
		</p>
		<div class="row mb-3">
			<label :for="formId" class="col-sm-4 col-form-label pt-0 pt-sm-2">
				{{ limitLabel }}
			</label>
			<div class="col-sm-8 col-lg-4 pe-0">
				<select
					:id="formId"
					v-model.number="selectedLimit"
					class="form-select form-select-sm mb-1"
					@change="changeLimit"
				>
					<option value="null">{{ $t("smartCost.none") }}</option>
					<option v-for="{ value, name } in limitOptions" :key="value" :value="value">
						{{ name }}
					</option>
				</select>
			</div>
			<div
				v-if="applyToAllVisible"
				class="col-sm-8 offset-sm-4 offset-lg-0 col-lg-4 d-flex align-items-baseline"
			>
				<div class="text-primary">{{ $t("smartCost.saved") }}</div>
				<button class="ms-1 btn btn-sm btn-link text-muted p-0" @click="applyToAll">
					{{ $t("smartCost.applyToAll") }}
				</button>
			</div>
		</div>
		<div class="justify-content-between mb-2 d-flex justify-content-between">
			<div class="text-start">
				<div class="label">
					{{ activeHoursLabel }}
				</div>
				<div class="value" :class="activeHoursClass">
					{{ activeHoursText }}
				</div>
			</div>
			<div class="text-end">
				<div class="label">
					<span v-if="activeSlot">{{ activeSlotName }}</span>
					<span v-else>{{ currentPriceLabel }}</span>
				</div>
				<div v-if="activeSlot" class="value text-primary">
					{{ activeSlotCost }}
				</div>
				<div v-else-if="activeSlots.length" class="value text-primary">
					{{ fmtActiveCostRange }}
				</div>
				<div v-else class="value value-inactive">
					{{ fmtTotalCostRange }}
				</div>
			</div>
		</div>
		<TariffChart
			v-if="rates.length"
			:slots="slots"
			@slot-hovered="slotHovered"
			@slot-selected="slotSelected"
		/>
	</div>
	<div v-else-if="currentLimit !== null">
		<div class="alert alert-warning">
			{{ resetWarningText }}
			<a href="#" class="text-warning" @click.prevent="resetLimit">
				{{ $t("smartCost.resetAction") }}
			</a>
		</div>
	</div>
</template>

<script lang="ts">
import formatter from "@/mixins/formatter";
import TariffChart from "./TariffChart.vue";
import { defineComponent, type PropType } from "vue";
import { type CURRENCY, type Rate, type SelectOption, type Slot } from "@/types/evcc";

type LimitDirection = "above" | "below";
type HighlightColor = "text-primary" | "text-warning";

export default defineComponent({
	name: "SmartTariffBase",
	components: { TariffChart },
	mixins: [formatter],
	props: {
		currentLimit: {
			type: [Number, null] as PropType<number | null>,
			required: true,
		},
		isCo2: Boolean,
		currency: String as PropType<CURRENCY>,
		applyAll: Boolean,
		possible: Boolean,
		tariff: Array,
		formId: String,
		title: String,
		description: String,
		limitLabel: String,
		optionsExtraHigh: Boolean,
		optionsStartAtZero: Boolean,
		activeHoursLabel: { type: String, required: true },
		currentPriceLabel: String,
		resetWarningText: String,
		limitDirection: { type: String as PropType<LimitDirection>, default: "below" },
		highlightColor: { type: String as PropType<HighlightColor>, default: "text-primary" },
		isSlotActive: {
			type: Function as PropType<(value: number | undefined) => boolean>,
			required: true,
		},
	},
	emits: ["save-limit", "delete-limit", "apply-to-all"],
	data() {
		return {
			selectedLimit: null as number | null,
			activeIndex: null as number | null,
			applyToAllVisible: false,
		};
	},
	computed: {
		rates(): Rate[] {
			if (this.tariff?.length) {
				return this.tariff.map((slot: any) => ({
					start: new Date(slot.start),
					end: new Date(slot.end),
					value: slot.value,
				}));
			}
			return [];
		},
		limitOptions(): SelectOption<number>[] {
			const { max } = this.optionsCostRange;

			const values = [] as number[];
			const stepSize = this.optionStepSize;
			for (let i = 1; i <= 100; i++) {
				const value = this.optionStartValue + stepSize * i;
				if (max !== undefined && value > max + stepSize) break;
				values.push(this.roundLimit(value) as number);
			}
			// add special entry if currently selected value is not in the scale
			const selected = this.selectedLimit;
			if (selected && !values.includes(selected)) {
				values.push(selected);
			}
			values.sort((a, b) => a - b);
			return values.map((value) => ({ value, name: this.formatLimit(value) }));
		},
		optionStartValue() {
			if (!this.rates?.length) {
				return 0;
			}
			const { min } = this.optionsCostRange;
			const stepSize = this.optionStepSize;
			// always show some negative values for price
			const start = this.optionsStartAtZero ? 0 : stepSize * -11;
			const minValue = min !== undefined ? Math.min(start, min) : start;
			return Math.floor(minValue / stepSize) * stepSize;
		},
		optionStepSize() {
			if (!this.rates?.length) {
				return 0.001;
			}
			const { min, max } = this.optionsCostRange;
			if (min === undefined || max === undefined) {
				return 0.001;
			}

			const baseSteps = [0.001, 0.002, 0.005];
			const range = max - Math.min(0, min);
			for (let scale = 1; scale <= 10000; scale *= 10) {
				for (const baseStep of baseSteps) {
					const step = baseStep * scale;
					if (range < step * 100) return step;
				}
			}
			return 1;
		},
		optionsCostRange() {
			const { min, max } = this.costRange(this.totalSlots);
			if (this.optionsExtraHigh && max) {
				return { min, max: max * 2 };
			}
			return { min, max };
		},
		slots(): Slot[] {
			if (!this.rates?.length) {
				return [];
			}

			const result = [] as Slot[];
			const rates = this.rates;
			const startTime = new Date();
			const oneHour = 3600000;

			for (let i = 0; i < 42; i++) {
				const start = new Date(startTime.getTime() + oneHour * i);
				const startHour = start.getHours();
				start.setMinutes(0);
				start.setSeconds(0);
				start.setMilliseconds(0);
				const end = new Date(start.getTime());
				end.setHours(startHour + 1);
				const endHour = end.getHours();
				const day = this.weekdayShort(start);
				const value = this.findRateInRange(start, end, rates)?.value;
				const active =
					this.limitDirection === "below" &&
					this.selectedLimit !== null &&
					value !== undefined &&
					value <= this.selectedLimit;
				const warning =
					this.limitDirection === "above" &&
					this.selectedLimit !== null &&
					value !== undefined &&
					value >= this.selectedLimit;
				const selectable = value !== undefined;
				result.push({
					day,
					value,
					startHour,
					endHour,
					charging: active,
					selectable,
					warning,
				});
			}

			return result;
		},
		totalSlots() {
			return this.slots.filter((s) => s.value !== undefined);
		},
		activeSlots() {
			return this.totalSlots.filter((s) => s.charging);
		},
		warningSlots() {
			return this.totalSlots.filter((s) => s.warning);
		},
		fmtTotalCostRange() {
			return this.fmtCostRange(this.costRange(this.totalSlots));
		},
		fmtActiveCostRange() {
			return this.fmtCostRange(this.costRange(this.activeSlots));
		},
		activeSlot(): Slot | null {
			return this.activeIndex !== null ? this.slots[this.activeIndex] : null;
		},
		activeSlotCost() {
			const value = this.activeSlot?.value;
			if (value === undefined) {
				return this.$t("main.targetChargePlan.unknownPrice");
			}
			return this.formatValue(value);
		},
		activeSlotName() {
			if (this.activeSlot) {
				const { day, startHour, endHour } = this.activeSlot;
				const range = `${startHour}–${endHour}`;
				return this.$t("main.targetChargePlan.timeRange", { day, range });
			}
			return null;
		},
		activeHoursClass() {
			if (this.limitDirection === "below") {
				return this.activeSlots.length ? "text-primary" : "value-inactive";
			}
			return this.warningSlots.length ? "text-warning" : "value-inactive";
		},
		activeHoursText() {
			const params = {
				active:
					this.limitDirection === "below"
						? this.activeSlots.length
						: this.warningSlots.length,
				total: this.totalSlots.length,
			};
			return this.$t("smartCost.activeHours", params);
		},
		limitOperator() {
			return this.limitDirection === "below" ? "≤" : "≥";
		},
	},
	watch: {
		currentLimit(limit) {
			this.selectedLimit = this.roundLimit(limit);
		},
	},
	mounted() {
		this.selectedLimit = this.roundLimit(this.currentLimit);
	},
	methods: {
		roundLimit(limit: number | null): number | null {
			return limit === null ? null : Math.round(limit * 1000) / 1000;
		},
		formatLimit(limit: number | null): string {
			if (limit === null) {
				return this.$t("smartCost.none");
			}
			return `${this.limitOperator} ${this.formatValue(limit)}`;
		},
		formatValue(value: number): string {
			if (this.isCo2) {
				return this.fmtCo2Medium(value);
			}
			return this.fmtPricePerKWh(value, this.currency);
		},

		findRateInRange(start: Date, end: Date, rates: Rate[]) {
			return rates.find((r) => {
				if (r.start.getTime() < start.getTime()) {
					return r.end.getTime() > start.getTime();
				}
				return r.start.getTime() < end.getTime();
			});
		},
		costRange(slots: Slot[]): { min: number | undefined; max: number | undefined } {
			let min = undefined as number | undefined;
			let max = undefined as number | undefined;
			slots.forEach((slot) => {
				if (slot.value === undefined) return;
				min = min === undefined ? slot.value : Math.min(min, slot.value);
				max = max === undefined ? slot.value : Math.max(max, slot.value);
			});
			return { min, max };
		},
		fmtCostRange({ min, max }: { min: number | undefined; max: number | undefined }): string {
			if (min === undefined || max === undefined) return "";
			const fmtMin = this.formatShortValue(min);
			const fmtMax = this.formatShortValue(max);
			return `${fmtMin} – ${fmtMax}`;
		},
		formatShortValue(value: number): string {
			if (this.isCo2) {
				return this.fmtCo2Short(value);
			}
			return this.fmtPricePerKWh(value, this.currency, true);
		},
		slotHovered(index: number) {
			this.activeIndex = index;
		},
		slotSelected(index: number) {
			const value = this.slots[index].value;
			if (value !== undefined) {
				// 3 decimal precision
				const valueRounded = Math.ceil(value * 1000) / 1000;
				this.selectedLimit = valueRounded;
				this.saveLimit(`${valueRounded}`);
			}
		},
		changeLimit($event: Event) {
			const value = ($event.target as HTMLSelectElement).value;
			if (value === "null") {
				this.resetLimit();
			} else {
				this.saveLimit(value);
			}
		},
		saveLimit(limit: string) {
			this.$emit("save-limit", limit);
			if (this.applyAll) {
				this.applyToAllVisible = true;
			}
		},
		resetLimit() {
			this.$emit("delete-limit");
			if (this.applyAll) {
				this.applyToAllVisible = true;
			}
		},
		applyToAll() {
			this.$emit("apply-to-all", this.selectedLimit);
			this.applyToAllVisible = false;
		},
	},
});
</script>

<style scoped>
.value {
	font-size: 18px;
	font-weight: bold;
}
.label {
	color: var(--evcc-gray);
	text-transform: uppercase;
}
.value-inactive {
	color: var(--evcc-gray);
}
</style>
