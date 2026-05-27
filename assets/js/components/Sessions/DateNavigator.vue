<template>
	<div
		class="d-sm-flex justify-content-lg-end gap-lg-4"
		:class="showDay || showMonth ? 'justify-content-between' : 'justify-content-end'"
	>
		<div v-if="showDay" class="d-flex justify-content-between">
			<DateNavigatorButton
				prev
				:disabled="!hasPrevDay"
				:highlight="highlightPrev"
				:onClick="emitPrevDay"
				data-testid="navigate-prev-day"
			/>
			<label class="day-picker position-relative d-block" role="button">
				<input
					type="date"
					class="day-picker-input"
					:value="dayIsoValue"
					:min="dayIsoMin"
					:max="dayIsoMax"
					data-testid="navigate-day"
					@change="
						onDateInputChange(
							($event.target as HTMLInputElement).value,
							$event.target as HTMLInputElement
						)
					"
				/>
				<button
					type="button"
					class="btn btn-sm border-0 text-truncate h-100"
					style="width: 11em"
					tabindex="-1"
				>
					{{ dayLongName }}
				</button>
			</label>
			<DateNavigatorButton
				next
				:disabled="!hasNextDay"
				:highlight="highlightNext"
				:onClick="emitNextDay"
				data-testid="navigate-next-day"
			/>
		</div>
		<div v-if="showMonth" class="d-none d-sm-flex justify-content-between">
			<DateNavigatorButton
				prev
				:disabled="!hasPrevMonth"
				:highlight="highlightPrev"
				:onClick="emitPrevMonth"
				data-testid="navigate-prev-month"
			/>
			<CustomSelect
				id="sessionsMonth"
				:options="monthOptions"
				:selected="month"
				@change="emitMonth($event.target.value)"
			>
				<button
					class="btn btn-sm border-0 text-truncate h-100"
					style="width: 8em"
					data-testid="navigate-month"
				>
					{{ monthName }}
				</button>
			</CustomSelect>
			<DateNavigatorButton
				next
				:disabled="!hasNextMonth"
				:highlight="highlightNext"
				:onClick="emitNextMonth"
				data-testid="navigate-next-month"
			/>
		</div>
		<div v-if="showMonth && !showDay" class="d-flex d-sm-none justify-content-between">
			<DateNavigatorButton
				prev
				:disabled="!hasPrevMonth"
				:highlight="highlightPrev"
				:onClick="emitPrevMonth"
				data-testid="navigate-prev-year-month"
			/>
			<CustomSelect
				id="sessionsMonthYear"
				:options="monthYearOptions"
				:selected="month"
				@change="emitMonthYear($event.target.value)"
			>
				<button
					class="btn btn-sm border-0 h-100 text-truncate"
					data-testid="navigate-month-year"
				>
					{{ monthYearName }}
				</button>
			</CustomSelect>
			<DateNavigatorButton
				next
				:disabled="!hasNextMonth"
				:highlight="highlightNext"
				:onClick="emitNextMonth"
				data-testid="navigate-next-year-month"
			/>
		</div>
		<div
			v-if="showYear"
			class="justify-content-between"
			:class="showMonth || showDay ? 'd-none d-sm-flex' : 'd-flex'"
		>
			<DateNavigatorButton
				prev
				:disabled="!hasPrevYear"
				:highlight="highlightPrev"
				:onClick="emitPrevYear"
				data-testid="navigate-prev-year"
			/>
			<CustomSelect
				id="sessionsYear"
				:options="yearOptions"
				:selected="year"
				@change="emitYear($event.target.value)"
			>
				<button
					class="btn btn-sm border-0 h-100"
					style="width: 4em"
					data-testid="navigate-year"
				>
					{{ year }}
				</button>
			</CustomSelect>
			<DateNavigatorButton
				next
				:disabled="!hasNextYear"
				:highlight="highlightNext"
				:onClick="emitNextYear"
				data-testid="navigate-next-year"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import DateNavigatorButton from "./DateNavigatorButton.vue";
import formatter from "@/mixins/formatter";
import type { SelectOption } from "@/types/evcc";
import { attachSwipeHandler } from "@/utils/swipe";

function daysInMonth(year: number, month: number) {
	return new Date(year, month, 0).getDate();
}

function dateOnly(d: Date) {
	return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

export default defineComponent({
	name: "DateNavigator",
	components: {
		CustomSelect,
		DateNavigatorButton,
	},
	mixins: [formatter],
	props: {
		day: { type: Number, default: 1 },
		month: { type: Number, required: true },
		year: { type: Number, required: true },
		startDate: { type: Date, required: true },
		showDay: Boolean,
		showMonth: Boolean,
		showYear: Boolean,
	},
	emits: ["update-date"],
	data() {
		return {
			detachSwipe: null as (() => void) | null,
			highlightPrev: false,
			highlightNext: false,
			highlightTimer: null as ReturnType<typeof setTimeout> | null,
		};
	},
	computed: {
		hasPrevDay() {
			const prev = new Date(this.year, this.month - 1, this.day - 1);
			return dateOnly(prev).getTime() >= dateOnly(this.startDate).getTime();
		},
		hasNextDay() {
			const next = new Date(this.year, this.month - 1, this.day + 1);
			return dateOnly(next).getTime() <= dateOnly(new Date()).getTime();
		},
		hasPrevMonth() {
			return (
				this.year > this.startDate.getFullYear() ||
				this.month > this.startDate.getMonth() + 1
			);
		},
		hasNextMonth() {
			const now = new Date();
			return this.year < now.getFullYear() || this.month < now.getMonth() + 1;
		},
		hasPrevYear() {
			return this.year > this.startDate.getFullYear();
		},
		hasNextYear() {
			return this.year < new Date().getFullYear();
		},
		dayOptions(): SelectOption<number>[] {
			const total = daysInMonth(this.year, this.month);
			return Array.from({ length: total }, (_, i) => i + 1).map((d) => ({
				name: this.fmtDayMonth(new Date(this.year, this.month - 1, d)),
				value: d,
			}));
		},
		monthOptions() {
			return Array.from({ length: 12 }, (_, i) => i + 1).map((month) => ({
				name: this.fmtMonth(new Date(this.year, month - 1, 1), false),
				value: month,
			}));
		},
		monthYearOptions() {
			const first = this.startDate;
			const last = new Date();
			const yearMonths = [];
			for (let year = first.getFullYear(); year <= last.getFullYear(); year++) {
				const startMonth = year === first.getFullYear() ? first.getMonth() + 1 : 1;
				const endMonth = year === last.getFullYear() ? last.getMonth() + 1 : 12;
				for (let month = startMonth; month <= endMonth; month++) {
					yearMonths.push({
						name: this.fmtMonthYear(new Date(year, month - 1, 1)),
						value: `${year}-${month}`,
					});
				}
			}
			return yearMonths;
		},
		yearOptions(): SelectOption<number>[] {
			const first = this.startDate;
			const last = new Date();
			const years = [];
			for (let year = first.getFullYear(); year <= last.getFullYear(); year++) {
				years.push({ name: year.toString(), value: year });
			}
			return years;
		},
		dayName() {
			return this.fmtDayMonth(new Date(this.year, this.month - 1, this.day));
		},
		dayLongName() {
			return this.fmtDayMonthYear(new Date(this.year, this.month - 1, this.day));
		},
		dayIsoValue() {
			const pad = (n: number) => String(n).padStart(2, "0");
			return `${this.year}-${pad(this.month)}-${pad(this.day)}`;
		},
		dayIsoMin() {
			const d = this.startDate;
			const pad = (n: number) => String(n).padStart(2, "0");
			return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
		},
		dayIsoMax() {
			const d = new Date();
			const pad = (n: number) => String(n).padStart(2, "0");
			return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
		},
		monthName() {
			const date = new Date();
			date.setMonth(this.month - 1, 1);
			return this.fmtMonth(date, false);
		},
		monthYearName() {
			const date = new Date();
			date.setMonth(this.month - 1, 1);
			date.setFullYear(this.year);
			return this.fmtMonthYear(date);
		},
	},
	mounted() {
		this.detachSwipe = attachSwipeHandler(document.body, {
			onSwipeLeft: () => this.swipeNext(),
			onSwipeRight: () => this.swipePrev(),
			ignoreSelector: "canvas, [_echarts_instance_]",
		});
	},
	unmounted() {
		this.detachSwipe?.();
		if (this.highlightTimer) clearTimeout(this.highlightTimer);
	},
	methods: {
		emitPrevDay() {
			const prev = new Date(this.year, this.month - 1, this.day - 1);
			this.$emit("update-date", {
				year: prev.getFullYear(),
				month: prev.getMonth() + 1,
				day: prev.getDate(),
			});
		},
		emitNextDay() {
			const next = new Date(this.year, this.month - 1, this.day + 1);
			this.$emit("update-date", {
				year: next.getFullYear(),
				month: next.getMonth() + 1,
				day: next.getDate(),
			});
		},
		emitDay(day: number) {
			const clamped = Math.min(Number(day), daysInMonth(this.year, this.month));
			this.$emit("update-date", { year: this.year, month: this.month, day: clamped });
		},
		onDateInputChange(value: string, target?: HTMLInputElement) {
			if (target) target.blur();
			if (!value) return;
			const [y, m, d] = value.split("-").map((s) => parseInt(s, 10));
			if (!y || !m || !d) return;
			this.$emit("update-date", { year: y, month: m, day: d });
		},
		clampDay(year: number, month: number) {
			return Math.min(this.day, daysInMonth(year, month));
		},
		emitPrevMonth() {
			const prev = new Date(this.year, this.month - 2, 1);
			const year = prev.getFullYear();
			const month = prev.getMonth() + 1;
			this.$emit("update-date", {
				year,
				month,
				...(this.showDay ? { day: this.clampDay(year, month) } : {}),
			});
		},
		emitNextMonth() {
			const next = new Date(this.year, this.month, 1);
			const year = next.getFullYear();
			const month = next.getMonth() + 1;
			this.$emit("update-date", {
				year,
				month,
				...(this.showDay ? { day: this.clampDay(year, month) } : {}),
			});
		},
		emitPrevYear() {
			const year = this.year - 1;
			this.$emit("update-date", {
				year,
				month: undefined,
				...(this.showDay ? { day: this.clampDay(year, this.month) } : {}),
			});
		},
		emitNextYear() {
			const year = this.year + 1;
			this.$emit("update-date", {
				year,
				month: undefined,
				...(this.showDay ? { day: this.clampDay(year, this.month) } : {}),
			});
		},
		emitMonth(month: number) {
			this.$emit("update-date", {
				year: this.year,
				month,
				...(this.showDay ? { day: this.clampDay(this.year, Number(month)) } : {}),
			});
		},
		emitMonthYear(monthYear: string) {
			const [yearStr, monthStr] = monthYear.split("-");
			const year = parseInt(yearStr || "0");
			const month = parseInt(monthStr || "0");
			this.$emit("update-date", {
				year,
				month,
				...(this.showDay ? { day: this.clampDay(year, month) } : {}),
			});
		},
		emitYear(year: number) {
			const y = Number(year);
			this.$emit("update-date", {
				year: y,
				month: undefined,
				...(this.showDay ? { day: this.clampDay(y, this.month) } : {}),
			});
		},
		flashHighlight(dir: "prev" | "next") {
			if (this.highlightTimer) clearTimeout(this.highlightTimer);
			this.highlightPrev = dir === "prev";
			this.highlightNext = dir === "next";
			this.highlightTimer = setTimeout(() => {
				this.highlightPrev = false;
				this.highlightNext = false;
			}, 300);
		},
		swipePrev() {
			if (this.showDay && this.hasPrevDay) this.emitPrevDay();
			else if (this.showMonth && this.hasPrevMonth) this.emitPrevMonth();
			else if (this.showYear && this.hasPrevYear) this.emitPrevYear();
			else return;
			this.flashHighlight("prev");
		},
		swipeNext() {
			if (this.showDay && this.hasNextDay) this.emitNextDay();
			else if (this.showMonth && this.hasNextMonth) this.emitNextMonth();
			else if (this.showYear && this.hasNextYear) this.emitNextYear();
			else return;
			this.flashHighlight("next");
		},
	},
});
</script>

<style scoped>
.btn {
	color: inherit;
}
.day-picker {
	border-radius: var(--bs-border-radius);
}
.day-picker:focus-within {
	outline: var(--bs-focus-ring-width) solid var(--bs-focus-ring-color);
	outline-offset: var(--bs-focus-ring-width);
}
.day-picker-input {
	position: absolute;
	inset: 0;
	width: 100%;
	height: 100%;
	opacity: 0;
	cursor: pointer;
	border: 0;
	padding: 0;
}
</style>
