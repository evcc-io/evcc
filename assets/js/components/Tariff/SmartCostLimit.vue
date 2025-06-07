<template>
	<div v-if="possible">
		<h6 v-if="isLoadpoint" class="mt-0">{{ title }}</h6>
		<p>
			{{ description }}
		</p>
		<div class="row mb-3">
			<label :for="formId" class="col-sm-4 col-form-label pt-0 pt-sm-2">
				{{ isCo2 ? $t("smartCost.co2Limit") : $t("smartCost.priceLimit") }}
			</label>
			<div class="col-sm-8 col-lg-4 pe-0">
				<select
					:id="formId"
					v-model.number="selectedSmartCostLimit"
					class="form-select form-select-sm mb-1"
					@change="changeSmartCostLimit"
				>
					<option value="null">{{ $t("smartCost.none") }}</option>
					<option v-for="{ value, name } in costOptions" :key="value" :value="value">
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
					{{ $t("smartCost.activeHoursLabel") }}
				</div>
				<div
					class="value"
					:class="chargingSlots.length ? 'text-primary' : 'value-inactive'"
				>
					{{
						$t("smartCost.activeHours", {
							total: totalSlots.length,
							charging: chargingSlots.length,
						})
					}}
				</div>
			</div>
			<div class="text-end">
				<div class="label">
					<span v-if="activeSlot">{{ activeSlotName }}</span>
					<span v-else-if="isCo2">{{ $t("smartCost.co2Label") }}</span>
					<span v-else>{{ $t("smartCost.priceLabel") }}</span>
				</div>
				<div v-if="activeSlot" class="value text-primary">
					{{ activeSlotCost }}
				</div>
				<div v-else-if="chargingSlots.length" class="value text-primary">
					{{ chargingCostRange }}
				</div>
				<div v-else class="value value-inactive">
					{{ totalCostRange }}
				</div>
			</div>
		</div>
		<TariffChart
			v-if="tariff"
			:slots="slots"
			@slot-hovered="slotHovered"
			@slot-selected="slotSelected"
		/>
	</div>
	<div v-else-if="smartCostLimit !== null">
		<div class="alert alert-warning">
			{{ $t("smartCost.resetWarning", { limit: fmtSmartCostLimit(smartCostLimit) }) }}
			<a href="#" class="text-warning" @click.prevent="saveSmartCostLimit('null')">
				{{ $t("smartCost.resetAction") }}
			</a>
		</div>
	</div>
</template>

<script lang="ts">
import formatter from "@/mixins/formatter";
import TariffChart from "./TariffChart.vue";
import { CO2_TYPE } from "@/units";
import api, { allowClientError } from "@/api";
import convertRates from "@/utils/convertRates";
import { defineComponent, type PropType } from "vue";
import type { Tariff, CURRENCY, Rate, SelectOption, Slot } from "@/types/evcc";

export default defineComponent({
	name: "SmartCostLimit",
	components: { TariffChart },
	mixins: [formatter],
	props: {
		smartCostLimit: {
			type: [Number, null] as PropType<number | null>,
			required: true,
		},
		smartCostType: String,
		tariffGrid: Number,
		currency: String as PropType<CURRENCY>,
		loadpointId: Number,
		multipleLoadpoints: Boolean,
		possible: Boolean,
	},
	data() {
		return {
			selectedSmartCostLimit: null as number | null,
			tariff: null as Tariff | null,
			startTime: null as Date | null,
			activeIndex: null as number | null,
			applyToAllVisible: false,
		};
	},
	computed: {
		isCo2(): boolean {
			return this.smartCostType === CO2_TYPE;
		},
		costOptions(): SelectOption<number>[] {
			const { max } = this.costRange(this.totalSlots);

			const values = [] as number[];
			const stepSize = this.optionStepSize;
			for (let i = 1; i <= 100; i++) {
				const value = this.optionStartValue + stepSize * i;
				if (max !== undefined && value > max + stepSize) break;
				values.push(this.roundLimit(value) as number);
			}
			// add special entry if currently selected value is not in the scale
			const selected = this.selectedSmartCostLimit;
			if (selected && !values.includes(selected)) {
				values.push(selected);
			}
			values.sort((a, b) => a - b);
			return values.map((value) => ({ value, name: this.fmtSmartCostLimit(value) }));
		},
		optionStartValue() {
			if (!this.tariff) {
				return 0;
			}
			const { min } = this.costRange(this.totalSlots);
			const stepSize = this.optionStepSize;
			// always show some negative values for price
			const start = this.isCo2 ? 0 : stepSize * -11;
			const minValue = min !== undefined ? Math.min(start, min) : start;
			return Math.floor(minValue / stepSize) * stepSize;
		},
		optionStepSize() {
			if (!this.tariff) {
				return 1;
			}
			const { min, max } = this.costRange(this.totalSlots);
			if (min === undefined || max === undefined) {
				return 1;
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
		slots(): Slot[] {
			const result: Slot[] = [];
			if (!this.tariff?.rates || !this.startTime) return result;

			const { rates } = this.tariff;
			const oneHour = 60 * 60 * 1000;

			for (let i = 0; i < 42; i++) {
				const start = new Date(this.startTime.getTime() + oneHour * i);
				const startHour = start.getHours();
				start.setMinutes(0);
				start.setSeconds(0);
				start.setMilliseconds(0);
				const end = new Date(start.getTime());
				end.setHours(startHour + 1);
				const endHour = end.getHours();
				const day = this.weekdayShort(start);
				// TODO: handle multiple matching time slots
				const value = this.findRateInRange(start, end, rates)?.value;
				const charging =
					this.selectedSmartCostLimit !== null &&
					value !== undefined &&
					value <= this.selectedSmartCostLimit;
				const selectable = value !== undefined;
				result.push({ day, value, startHour, endHour, charging, selectable });
			}

			return result;
		},
		totalSlots() {
			return this.slots.filter((s) => s.value !== undefined);
		},
		chargingSlots() {
			return this.totalSlots.filter((s) => s.charging);
		},
		totalCostRange() {
			return this.fmtCostRange(this.costRange(this.totalSlots));
		},
		chargingCostRange() {
			return this.fmtCostRange(this.costRange(this.chargingSlots));
		},
		activeSlot(): Slot | null {
			return this.activeIndex !== null ? this.slots[this.activeIndex] : null;
		},
		activeSlotName() {
			if (this.activeSlot) {
				const { day, startHour, endHour } = this.activeSlot;
				const range = `${startHour}–${endHour}`;
				return this.$t("main.targetChargePlan.timeRange", { day, range });
			}
			return null;
		},
		activeSlotCost() {
			const value = this.activeSlot?.value;
			if (value === undefined) {
				return this.$t("main.targetChargePlan.unknownPrice");
			}
			if (this.isCo2) {
				return this.fmtCo2Medium(value);
			}
			return this.fmtPricePerKWh(value, this.currency);
		},
		title() {
			return this.$t(`smartCost.${this.isCo2 ? "clean" : "cheap"}Title`);
		},
		description() {
			return this.$t(`smartCost.${this.loadpointId ? "loadpoint" : "battery"}Description`);
		},
		formId() {
			return `smartCostLimit-${this.loadpointId || "battery"}`;
		},
		isLoadpoint() {
			return !!this.loadpointId;
		},
	},
	watch: {
		tariffGrid() {
			this.updateTariff();
		},
		smartCostLimit(limit) {
			this.selectedSmartCostLimit = this.roundLimit(limit);
		},
	},
	mounted() {
		this.updateTariff();
		this.selectedSmartCostLimit = this.roundLimit(this.smartCostLimit);
	},
	methods: {
		roundLimit(limit: number | null): number | null {
			return limit === null ? null : Math.round(limit * 1000) / 1000;
		},
		fmtSmartCostLimit(limit: number | null): string {
			if (limit === null) {
				return this.$t("smartCost.none");
			}
			return `≤ ${
				this.isCo2 ? this.fmtCo2Medium(limit) : this.fmtPricePerKWh(limit, this.currency)
			}`;
		},
		async updateTariff() {
			try {
				const res = await api.get(`tariff/planner`, allowClientError);
				if (res.status === 200) {
					this.tariff = {
						rates: convertRates(res.data.result.rates),
						lastUpdate: new Date(),
					};
					this.startTime = new Date();
				}
			} catch (e) {
				console.error(e);
			}
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
			const fmtMin = this.isCo2
				? this.fmtCo2Short(min)
				: this.fmtPricePerKWh(min, this.currency, true);
			const fmtMax = this.isCo2
				? this.fmtCo2Short(max)
				: this.fmtPricePerKWh(max, this.currency, true);
			return `${fmtMin} – ${fmtMax}`;
		},
		slotHovered(index: number) {
			this.activeIndex = index;
		},
		slotSelected(index: number) {
			const value = this.slots[index].value;
			if (value !== undefined) {
				// 3 decimal precision
				const valueRounded = Math.ceil(value * 1000) / 1000;
				this.saveSmartCostLimit(`${valueRounded}`);
			}
		},
		changeSmartCostLimit($event: Event) {
			const value = ($event.target as HTMLSelectElement).value;
			this.saveSmartCostLimit(value);
		},
		async saveSmartCostLimit(limit: string) {
			const url = this.isLoadpoint
				? `loadpoints/${this.loadpointId}/smartcostlimit`
				: "batterygridchargelimit";

			// delete
			try {
				if (limit === "null") {
					await api.delete(url);
				} else {
					await api.post(`${url}/${encodeURIComponent(limit)}`);
				}
				if (this.isLoadpoint && this.multipleLoadpoints) {
					this.applyToAllVisible = true;
				}
			} catch (err) {
				console.error(err);
			}
		},
		async applyToAll() {
			try {
				if (this.selectedSmartCostLimit === null) {
					await api.delete("smartcostlimit");
				} else {
					await api.post(
						`smartcostlimit/${encodeURIComponent(this.selectedSmartCostLimit)}`
					);
				}
				this.applyToAllVisible = false;
			} catch (err) {
				console.error(err);
			}
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
