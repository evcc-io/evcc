<template>
	<div>
		<h4 class="text-evcc mb-3">{{ title }}</h4>
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
					<option value="0">{{ $t("smartCost.none") }}</option>
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
</template>

<script>
import formatter from "../mixins/formatter";
import TariffChart from "./TariffChart.vue";
import { CO2_TYPE } from "../units";
import api from "../api";

export default {
	name: "SmartCostLimit",
	components: { TariffChart },
	mixins: [formatter],
	props: {
		smartCostLimit: { type: Number, default: 0 },
		smartCostType: String,
		tariffGrid: Number,
		currency: String,
		loadpointId: Number,
		multipleLoadpoints: Boolean,
	},
	data: function () {
		return {
			selectedSmartCostLimit: 0,
			tariff: null,
			startTime: null,
			activeIndex: null,
			applyToAllVisible: false,
		};
	},
	computed: {
		isCo2() {
			return this.smartCostType === CO2_TYPE;
		},
		costOptions() {
			const values = [];
			const stepSize = this.optionStepSize;
			for (let i = 1; i <= 100; i++) {
				const value = this.optionStartValue + stepSize * i;
				if (value != 0) {
					values.push(value);
				}
			}
			// add special entry if currently selected value is not in the scale
			const selected = this.selectedSmartCostLimit;
			if (selected !== undefined && !values.includes(selected)) {
				values.push(selected);
			}
			values.sort((a, b) => a - b);
			return values.map((value) => {
				const name = `≤ ${
					this.isCo2
						? this.fmtCo2Medium(value)
						: this.fmtPricePerKWh(value, this.currency)
				}`;
				return { value, name };
			});
		},
		optionStartValue() {
			if (!this.tariff) {
				return 0;
			}
			const { min } = this.costRange(this.totalSlots);
			const minValue = Math.min(0, min);
			const stepSize = this.optionStepSize;
			return Math.ceil(minValue / stepSize) * stepSize;
		},
		optionStepSize() {
			if (!this.tariff) {
				return 1;
			}
			const { min, max } = this.costRange(this.totalSlots);
			for (const scale of [0.1, 1, 10, 50, 100, 200, 500, 1000, 2000, 5000, 10000]) {
				if (max - Math.min(0, min) < scale) {
					return scale / 100;
				}
			}
			return 1;
		},
		slots() {
			const result = [];
			if (!this.tariff?.rates) return result;

			const rates = this.convertDates(this.tariff.rates);
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
				const price = this.findSlotInRange(start, end, rates)?.price;
				const charging =
					price <= this.selectedSmartCostLimit && this.selectedSmartCostLimit !== 0;
				const selectable = price !== undefined;
				result.push({ day, price, startHour, endHour, charging, selectable });
			}

			return result;
		},
		totalSlots() {
			return this.slots.filter((s) => s.price !== undefined);
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
		activeSlot() {
			return this.slots[this.activeIndex];
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
			const price = this.activeSlot?.price;
			if (price === undefined) {
				return this.$t("main.targetChargePlan.unknownPrice");
			}
			if (this.isCo2) {
				return this.fmtCo2Medium(price);
			}
			return this.fmtPricePerKWh(price, this.currency);
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
	},
	watch: {
		tariffGrid() {
			this.updateTariff();
		},
		smartCostLimit(limit) {
			this.selectedSmartCostLimit = limit;
		},
	},
	mounted() {
		this.updateTariff();
		this.selectedSmartCostLimit = this.smartCostLimit;
	},
	methods: {
		updateTariff: async function () {
			try {
				this.tariff = (await api.get(`tariff/planner`)).data.result;
				this.startTime = new Date();
			} catch (e) {
				console.error(e);
			}
		},
		convertDates(list) {
			if (!list?.length) {
				return [];
			}
			return list.map((item) => {
				return {
					start: new Date(item.start),
					end: new Date(item.end),
					price: item.price,
				};
			});
		},
		findSlotInRange(start, end, slots) {
			return slots.find((s) => {
				if (s.start.getTime() < start.getTime()) {
					return s.end.getTime() > start.getTime();
				}
				return s.start.getTime() < end.getTime();
			});
		},
		costRange(slots) {
			let min = undefined;
			let max = undefined;
			slots.forEach((slot) => {
				min = min === undefined ? slot.price : Math.min(min, slot.price);
				max = max === undefined ? slot.price : Math.max(max, slot.price);
			});
			return { min, max };
		},
		fmtCostRange({ min, max }) {
			const fmtMin = this.isCo2
				? this.fmtCo2Short(min)
				: this.fmtPricePerKWh(min, this.currency, true);
			const fmtMax = this.isCo2
				? this.fmtCo2Short(max)
				: this.fmtPricePerKWh(max, this.currency, true);
			return `${fmtMin} – ${fmtMax}`;
		},
		slotHovered(index) {
			this.activeIndex = index;
		},
		setSelectedSmartCostLimit(limit) {
			if (limit === 0) {
				this.selectedSmartCostLimit = 0;
				return;
			}
			const nextOption = this.costOptions.find(({ value }) => value >= limit);
			if (nextOption) {
				this.selectedSmartCostLimit = nextOption.value;
			}
		},
		slotSelected(index) {
			const price = this.slots[index].price;
			if (price !== undefined) {
				this.setSelectedSmartCostLimit(price);
				this.saveSmartCostLimit(this.selectedSmartCostLimit);
			}
		},
		changeSmartCostLimit($event) {
			this.saveSmartCostLimit($event.target.value);
		},
		async saveSmartCostLimit(limit) {
			const isLoadpoint = !!this.loadpointId;
			const url = isLoadpoint
				? `loadpoints/${this.loadpointId}/smartcostlimit`
				: "batterysmartcostlimit"; // currently not implemented
			try {
				await api.post(`${url}/${encodeURIComponent(limit)}`);
				if (isLoadpoint && this.multipleLoadpoints) {
					this.applyToAllVisible = true;
				}
			} catch (err) {
				console.error(err);
			}
		},
		async applyToAll() {
			try {
				await api.post(`smartcostlimit/${encodeURIComponent(this.selectedSmartCostLimit)}`);
				this.applyToAllVisible = false;
			} catch (err) {
				console.error(err);
			}
		},
	},
};
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
