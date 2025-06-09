<template>
	<div
		class="d-sm-flex justify-content-lg-end gap-lg-4"
		:class="showMonth ? 'justify-content-between' : 'justify-content-end'"
	>
		<div v-if="showMonth" class="d-none d-sm-flex justify-content-between">
			<DateNavigatorButton
				prev
				:disabled="!hasPrevMonth"
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
				:onClick="emitNextMonth"
				data-testid="navigate-next-month"
			/>
		</div>
		<div v-if="showMonth" class="d-flex d-sm-none justify-content-between">
			<DateNavigatorButton
				prev
				:disabled="!hasPrevMonth"
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
				:onClick="emitNextMonth"
				data-testid="navigate-next-year-month"
			/>
		</div>
		<div
			v-if="showYear"
			class="justify-content-between"
			:class="showMonth ? 'd-none d-sm-flex' : 'd-flex'"
		>
			<DateNavigatorButton
				prev
				:disabled="!hasPrevYear"
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

export default defineComponent({
	name: "DateNavigator",
	components: {
		CustomSelect,
		DateNavigatorButton,
	},
	mixins: [formatter],
	props: {
		month: { type: Number, required: true },
		year: { type: Number, required: true },
		startDate: { type: Date, required: true },
		showMonth: Boolean,
		showYear: Boolean,
	},
	emits: ["update-date"],
	computed: {
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
	methods: {
		emitPrevMonth() {
			const prevMonthDate = new Date(this.year, this.month - 2, 1);
			this.$emit("update-date", {
				year: prevMonthDate.getFullYear(),
				month: prevMonthDate.getMonth() + 1,
			});
		},
		emitNextMonth() {
			const nextMonthDate = new Date(this.year, this.month, 1);
			this.$emit("update-date", {
				year: nextMonthDate.getFullYear(),
				month: nextMonthDate.getMonth() + 1,
			});
		},
		emitPrevYear() {
			this.$emit("update-date", { year: this.year - 1, month: undefined });
		},
		emitNextYear() {
			this.$emit("update-date", { year: this.year + 1, month: undefined });
		},
		emitMonth(month: number) {
			this.$emit("update-date", { year: this.year, month });
		},
		emitMonthYear(monthYear: string) {
			const [year, month] = monthYear.split("-");
			this.$emit("update-date", { year: parseInt(year), month: parseInt(month) });
		},
		emitYear(year: number) {
			this.$emit("update-date", { year, month: undefined });
		},
	},
});
</script>

<style scoped>
.btn {
	color: inherit;
}
</style>
