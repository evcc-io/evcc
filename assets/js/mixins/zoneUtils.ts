import { defineComponent } from "vue";
import formatter from "./formatter";

const ZONE_MONTH_CODES = [
	"jan",
	"feb",
	"mar",
	"apr",
	"may",
	"jun",
	"jul",
	"aug",
	"sep",
	"oct",
	"nov",
	"dec",
];
const ZONE_DAY_CODES = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"];

export default defineComponent({
	mixins: [formatter],
	methods: {
		parseWeekdaysString(daysStr: string): number[] {
			const weekdays = [];
			for (const part of daysStr.split(",")) {
				const index = ZONE_DAY_CODES.indexOf(part);
				if (index !== -1) {
					weekdays.push(index);
				}
			}
			return weekdays;
		},
		parseMonthsString(monthsStr: string): number[] {
			const months = [];
			for (const part of monthsStr.split(",")) {
				const index = ZONE_MONTH_CODES.indexOf(part);
				if (index !== -1) {
					months.push(index);
				}
			}
			return months;
		},
		formatWeekdaysToString(weekdays: number[]): string {
			if (!weekdays || weekdays.length === 0) return "";
			return weekdays.map((d) => ZONE_DAY_CODES[d]).join(",");
		},
		formatMonthsToString(months: number[]): string {
			if (!months || months.length === 0) return "";
			return months.map((m) => ZONE_MONTH_CODES[m]).join(",");
		},
		weekdaysLabel(weekdays: number[]): string {
			if (!weekdays || weekdays.length === 0 || weekdays.length === 7) {
				return this.$t("config.tariff.zones.allDays");
			}
			return this.getShortenedWeekdaysLabel(weekdays);
		},
		monthsLabel(months: number[]): string {
			if (!months || months.length === 0 || months.length === 12) {
				return this.$t("config.tariff.zones.allMonths");
			}
			return this.getShortenedMonthsLabel(months);
		},
	},
});
