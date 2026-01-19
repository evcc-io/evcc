<template>
	<div class="d-flex align-items-center justify-content-between py-2 ps-3 pe-2 border rounded">
		<div class="flex-grow-1">
			<span class="fw-semibold">{{ formattedPrice }}</span>
			<small class="text-muted ms-2">{{ formattedConstraints }}</small>
		</div>
		<div class="d-flex">
			<button
				type="button"
				class="btn btn-sm btn-outline-secondary border-0"
				:aria-label="$t('config.tariff.zones.edit')"
				@click="$emit('edit')"
			>
				<shopicon-regular-edit size="s" class="flex-shrink-0"></shopicon-regular-edit>
			</button>
			<button
				v-if="showDelete"
				type="button"
				class="btn btn-sm btn-outline-secondary border-0"
				:aria-label="$t('config.tariff.zones.remove')"
				@click="$emit('remove')"
			>
				<shopicon-regular-trash size="s" class="flex-shrink-0"></shopicon-regular-trash>
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import { type PropType } from "vue";
import "@h2d2/shopicons/es/regular/trash";
import "@h2d2/shopicons/es/regular/edit";
import formatter from "@/mixins/formatter";
import { CURRENCY, type Zone } from "@/types/evcc";
import { ZONE_MONTH_CODES, ZONE_DAY_CODES } from "@/utils/zoneConstants";

export default {
	name: "PropertyZoneSummary",
	mixins: [formatter],
	props: {
		zone: { type: Object as PropType<Zone>, required: true },
		currency: { type: String as PropType<CURRENCY>, required: true },
		showDelete: { type: Boolean, required: true },
	},
	emits: ["edit", "remove"],
	computed: {
		displayFactor() {
			return this.pricePerKWhDisplayFactor(this.currency);
		},
		formattedPrice() {
			if (this.zone.price == null) return "—";
			const displayPrice = this.zone.price * this.displayFactor;
			const roundedPrice = Math.round(displayPrice * 1e6) / 1e6;
			const shortUnit = this.pricePerKWhUnit(this.currency, true);
			return `${roundedPrice.toFixed(2)} ${shortUnit}`;
		},
		formattedConstraints() {
			const parts = [];
			if (this.zone.days) {
				const weekdays = this.parseWeekdaysString(this.zone.days);
				if (weekdays.length > 0) {
					parts.push(this.getShortenedWeekdaysLabel(weekdays));
				}
			}
			if (this.zone.months) {
				const months = this.parseMonthsString(this.zone.months);
				if (months.length > 0) {
					parts.push(this.monthsLabel(months));
				}
			}
			if (this.zone.hours) {
				parts.push(this.zone.hours);
			}
			return parts.length > 0 ? parts.join(", ") : this.$t("config.tariff.zones.allTimes");
		},
	},
	methods: {
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
		monthsLabel(months: number[]) {
			if (!months || months.length === 0 || months.length === 12) {
				return this.$t("config.tariff.zones.allMonths");
			}
			return this.getShortenedMonthsLabel(months);
		},
	},
};
</script>
