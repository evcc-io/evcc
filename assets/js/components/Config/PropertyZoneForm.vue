<template>
	<div class="border rounded p-3">
		<!-- Price input -->
		<div class="mb-3">
			<label class="form-label">{{ $t("config.tariff.zones.price") }}</label>
			<div class="d-flex w-50 w-min-200">
				<input
					v-model="uiZone.price"
					type="number"
					step="any"
					class="form-control text-end"
					:class="{ 'is-invalid': isPriceInvalid }"
					style="border-top-right-radius: 0; border-bottom-right-radius: 0"
					:placeholder="$t('config.tariff.zones.pricePlaceholder')"
				/>
				<span
					class="input-group-text"
					:class="{ 'border-danger': isPriceInvalid }"
					style="border-top-left-radius: 0; border-bottom-left-radius: 0"
					>{{ priceUnit }}</span
				>
			</div>
			<div v-if="isPriceInvalid" class="invalid-feedback d-block">
				{{ $t("config.tariff.zones.priceRequired") }}
			</div>
		</div>

		<!-- Month selector -->
		<div class="mb-3">
			<label class="form-label">
				{{ $t("config.tariff.zones.months") }}
				<small class="evcc-gray">{{ $t("config.form.optional") }}</small>
			</label>
			<MultiSelect
				:id="`zone-months-${index}`"
				v-model="uiZone.months"
				:options="monthOptions"
				:selectAllLabel="$t('main.chargingPlan.selectAll')"
			>
				{{ monthsLabel(uiZone.months) }}
			</MultiSelect>
		</div>

		<!-- Weekday selector -->
		<div class="mb-3">
			<label class="form-label">
				{{ $t("config.tariff.zones.weekdays") }}
				<small class="evcc-gray">{{ $t("config.form.optional") }}</small>
			</label>
			<MultiSelect
				:id="`zone-weekdays-${index}`"
				v-model="uiZone.weekdays"
				:options="dayOptions"
				:selectAllLabel="$t('main.chargingPlan.selectAll')"
			>
				{{ weekdaysLabel(uiZone.weekdays) }}
			</MultiSelect>
		</div>

		<!-- Time range -->
		<div class="mb-3">
			<label class="form-label">
				{{ $t("config.tariff.zones.hours") }}
				<small class="evcc-gray">{{ $t("config.form.optional") }}</small>
			</label>
			<div class="d-flex gap-2 align-items-center">
				<input
					v-model="uiZone.timeFrom"
					type="time"
					class="form-control"
					:class="{ 'is-invalid': isTimeRangeInvalid }"
				/>
				<span>–</span>
				<input
					v-model="uiZone.timeTo"
					type="time"
					class="form-control"
					:class="{ 'is-invalid': isTimeRangeInvalid }"
				/>
			</div>
			<div v-if="isTimeRangeInvalid" class="invalid-feedback d-block">
				{{ $t("config.tariff.zones.timeRangeError") }}
			</div>
		</div>

		<!-- Actions -->
		<div class="d-flex justify-content-between align-items-center">
			<button type="button" class="btn btn-link text-muted" @click="$emit('cancel')">
				{{ $t("config.tariff.zones.cancel") }}
			</button>
			<button type="button" class="btn btn-primary" @click="handleSave">
				{{ $t("config.tariff.zones.save") }}
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import { type PropType } from "vue";
import formatter from "@/mixins/formatter";
import MultiSelect from "../Helper/MultiSelect.vue";
import { CURRENCY, type Zone } from "@/types/evcc";
import { ZONE_MONTH_CODES, ZONE_DAY_CODES } from "@/utils/zoneConstants";

type UiZone = {
	price: number | null;
	weekdays: number[];
	months: number[];
	timeFrom: string;
	timeTo: string;
};

export default {
	name: "PropertyZoneForm",
	components: { MultiSelect },
	mixins: [formatter],
	props: {
		zone: { type: Object as PropType<Zone>, required: true },
		currency: { type: String as PropType<CURRENCY>, required: true },
		index: { type: Number, required: true },
	},
	emits: ["update:zone", "save", "cancel"],
	data() {
		return {
			uiZone: {} as UiZone,
			saveAttempted: false,
		};
	},
	computed: {
		displayFactor() {
			return this.pricePerKWhDisplayFactor(this.currency);
		},
		priceUnit() {
			return this.pricePerKWhUnit(this.currency);
		},
		dayOptions() {
			return this.getWeekdaysList("long");
		},
		monthOptions() {
			return this.getMonthsList("long");
		},
		validatePrice() {
			const price = this.uiZone.price;
			return price === null || isNaN(price);
		},
		validateTimeRange() {
			const { timeFrom, timeTo } = this.uiZone;
			if (timeFrom === "00:00" && timeTo === "00:00") return false;
			if (!timeFrom || !timeTo) return false;
			if (timeTo === "00:00" && timeFrom !== "00:00") return false;
			return timeFrom >= timeTo;
		},
		isTimeRangeInvalid() {
			if (!this.saveAttempted) return false;
			return this.validateTimeRange;
		},
		isPriceInvalid() {
			if (!this.saveAttempted) return false;
			return this.validatePrice;
		},
		hasValidationErrors() {
			return this.validatePrice || this.validateTimeRange;
		},
	},
	watch: {
		zone: {
			handler(newZone: Zone) {
				this.uiZone = this.convertToUiZone(newZone);
			},
			immediate: true,
		},
	},
	methods: {
		convertToUiZone(zone: Zone) {
			const [timeFrom, timeTo] = zone.hours.split("-");
			return {
				price: zone.price,
				weekdays: this.parseWeekdaysString(zone.days),
				months: this.parseMonthsString(zone.months),
				timeFrom: timeFrom || "00:00",
				timeTo: timeTo || "00:00",
			};
		},
		convertFromUiZone(uiZone: UiZone) {
			const isAllDay = uiZone.timeFrom === "00:00" && uiZone.timeTo === "00:00";
			return {
				price: uiZone.price != null ? uiZone.price / this.displayFactor : null,
				days: this.formatWeekdaysToString(uiZone.weekdays),
				months: this.formatMonthsToString(uiZone.months),
				hours: isAllDay ? "" : `${uiZone.timeFrom}-${uiZone.timeTo}`,
			};
		},
		parseWeekdaysString(daysStr: string) {
			const weekdays = [];
			for (const part of daysStr.split(",")) {
				const index = ZONE_DAY_CODES.indexOf(part);
				if (index !== -1) {
					weekdays.push(index);
				}
			}
			return weekdays;
		},
		parseMonthsString(monthsStr: string) {
			const months = [];
			for (const part of monthsStr.split(",")) {
				const index = ZONE_MONTH_CODES.indexOf(part);
				if (index !== -1) {
					months.push(index);
				}
			}
			return months;
		},
		formatWeekdaysToString(weekdays: number[]) {
			if (!weekdays || weekdays.length === 0) return "";
			return weekdays.map((d) => ZONE_DAY_CODES[d]).join(",");
		},
		formatMonthsToString(months: number[]) {
			if (!months || months.length === 0) return "";
			return months.map((m) => ZONE_MONTH_CODES[m]).join(",");
		},
		weekdaysLabel(weekdays: number[]) {
			if (!weekdays || weekdays.length === 0 || weekdays.length === 7) {
				return this.$t("config.tariff.zones.allDays");
			}
			return this.getShortenedWeekdaysLabel(weekdays);
		},
		monthsLabel(months: number[]) {
			if (!months || months.length === 0 || months.length === 12) {
				return this.$t("config.tariff.zones.allMonths");
			}
			return this.getShortenedMonthsLabel(months);
		},
		handleSave() {
			this.saveAttempted = true;
			if (!this.hasValidationErrors) {
				const convertedZone = this.convertFromUiZone(this.uiZone);
				this.$emit("save", convertedZone);
			}
		},
	},
};
</script>

<style scoped>
.w-min-200 {
	min-width: min(200px, 100%);
}

/* Hide spinner for number input */
input[type="number"]::-webkit-inner-spin-button,
input[type="number"]::-webkit-outer-spin-button {
	-webkit-appearance: none;
	appearance: none;
	margin: 0;
}

input[type="number"] {
	-moz-appearance: textfield;
	appearance: textfield;
}
</style>
