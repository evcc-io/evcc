<template>
	<div v-if="possible">
		<h6 v-if="title" class="mt-0">{{ title }}</h6>
		<p>
			{{ description }}
		</p>
		<div class="row mb-3 align-items-center">
			<label :for="formId" class="col-sm-4 col-form-label pt-0 pt-sm-2">
				{{ limitLabel }}
			</label>
			<div class="col-sm-8 col-lg-4 pe-0">
				<div class="input-group input-group-sm mb-1 mb-lg-0">
					<div class="input-group-text">
						<div class="form-check form-switch m-0">
							<input
								:id="formId + 'Active'"
								:checked="active"
								class="form-check-input"
								type="checkbox"
								role="switch"
								:aria-label="$t('smartCost.enable')"
								@change="toggleActive"
							/>
						</div>
					</div>
					<select
						:id="formId"
						v-model.number="selectedLimit"
						class="form-select form-select-sm"
						:disabled="!active"
						:aria-label="limitLabel"
						@change="changeLimit"
					>
						<option v-for="{ value, name } in limitOptions" :key="value" :value="value">
							{{ name }}
						</option>
					</select>
				</div>
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
			<div class="text-start" data-testid="active-hours">
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
		lastLimit: { type: Number, default: 0 },
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
			active: false,
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
			for (let i = 0; i <= 100; i++) {
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

			const rates = this.rates;
			const quarterHour = 15 * 60 * 1000;

			const base = new Date();
			base.setSeconds(0, 0);
			base.setMinutes(base.getMinutes() - (base.getMinutes() % 15));

			return Array.from({ length: 96 * 4 }, (_, i) => {
				const start = new Date(base.getTime() + quarterHour * i);
				const end = new Date(start.getTime() + quarterHour);
				const value = this.findRateInRange(start, end, rates)?.value;
				const active =
					this.limitDirection === "below" &&
					this.currentLimit !== null &&
					value !== undefined &&
					value <= this.currentLimit;
				const warning =
					this.limitDirection === "above" &&
					this.currentLimit !== null &&
					value !== undefined &&
					value >= this.currentLimit;

				return {
					day: this.weekdayShort(start),
					value,
					start,
					end,
					charging: active,
					selectable: value !== undefined,
					warning,
				};
			});
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
			return this.activeIndex !== null ? this.slots[this.activeIndex] || null : null;
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
				const { day, start, end } = this.activeSlot;
				const range = `${this.fmtTimeString(start)}–${this.fmtTimeString(end)}`;
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
			const active =
				this.limitDirection === "below"
					? this.activeSlots.length
					: this.warningSlots.length;
			return this.fmtDurationLong(active * 15 * 60, "short");
		},
		limitOperator() {
			return this.limitDirection === "below" ? "≤" : "≥";
		},
	},
	watch: {
		currentLimit() {
			this.initLimit();
		},
	},
	mounted() {
		this.initLimit();
	},
	methods: {
		roundLimit(limit: number | null): number | null {
			return limit === null ? null : Math.round(limit * 1000) / 1000;
		},
		initLimit() {
			if (this.currentLimit === null) {
				this.active = false;
				this.selectedLimit = this.lastLimit;
			} else {
				this.active = true;
				this.selectedLimit = this.roundLimit(this.currentLimit);
			}
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
			const value = this.slots[index]?.value;
			if (value !== undefined) {
				// 3 decimal precision
				const valueRounded = Math.ceil(value * 1000) / 1000;
				this.selectedLimit = valueRounded;
				this.active = true;
				this.saveLimit(valueRounded);
			}
		},
		changeLimit($event: Event) {
			const value = parseFloat(($event.target as HTMLSelectElement).value);
			this.saveLimit(value);
		},
		toggleActive($event: Event) {
			const active = ($event.target as HTMLInputElement).checked;
			if (active) {
				this.saveLimit(this.lastLimit);
			} else {
				this.resetLimit();
			}
			this.active = active;
			if (this.applyAll) {
				this.applyToAllVisible = true;
			}
		},
		saveLimit(limit: number) {
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
			this.$emit("apply-to-all", this.currentLimit);
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
