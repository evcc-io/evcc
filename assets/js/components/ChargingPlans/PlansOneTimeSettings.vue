<template>
	<div v-for="(plan, index) in rows" :key="plan.key" class="row mb-4" data-testid="plan-entry">
		<div class="plan-id d-none d-lg-flex align-items-center fs-6">#{{ index + 2 }}</div>
		<div class="col-5 col-lg-3 mb-2 mb-lg-0">
			<input
				:id="formId(index, 'day')"
				:value="plan.day"
				:min="today"
				type="date"
				class="form-control"
				required
				data-testid="one-time-plan-day"
				@change="update(index, 'day', $event)"
			/>
		</div>
		<div class="col-4 col-lg-2 mb-2 mb-lg-0">
			<input
				:id="formId(index, 'time')"
				:value="plan.time"
				type="time"
				class="form-control"
				required
				data-testid="one-time-plan-time"
				@change="update(index, 'time', $event)"
			/>
		</div>
		<div class="col-3 mb-2 mb-lg-0">
			<select
				:id="formId(index, 'soc')"
				:value="plan.soc"
				class="form-select"
				data-testid="one-time-plan-soc"
				@change="update(index, 'soc', $event)"
			>
				<option v-for="opt in socOptions" :key="opt.value" :value="opt.value">
					{{ opt.name }}
				</option>
			</select>
		</div>
		<div class="col-12 col-lg-2 d-flex align-items-center">
			<button
				type="button"
				class="btn btn-sm btn-outline-secondary border-0 text-decoration-underline"
				data-testid="one-time-plan-remove"
				@click="remove(index)"
			>
				{{ $t("main.chargingPlan.remove") }}
			</button>
		</div>
	</div>
	<div class="d-flex align-items-center pb-4">
		<button
			type="button"
			class="add-plan d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
			data-testid="one-time-plan-add"
			@click="add"
		>
			<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
			{{ $t("main.chargingPlan.addOneTimePlan") }}
		</button>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/plus";
import formatter from "@/mixins/formatter";
import { distanceUnit } from "@/units";
import { defineComponent, type PropType } from "vue";
import type { OneTimePlan, SelectOption } from "@/types/evcc";

const DEFAULT_TARGET_TIME = "07:00";
const DEFAULT_TARGET_SOC = 80;

export default defineComponent({
	name: "ChargingPlansOneTimeSettings",
	mixins: [formatter],
	props: {
		id: [Number, String],
		rangePerSoc: Number,
		plans: { type: Array as PropType<OneTimePlan[]>, default: () => [] },
	},
	emits: ["updated"],
	computed: {
		today(): string {
			return this.fmtDayString(new Date());
		},
		rows() {
			return this.plans.map(({ time, soc }) => {
				const date = new Date(time);
				return {
					key: time,
					day: this.fmtDayString(date),
					time: this.fmtTimeString(date),
					soc,
				};
			});
		},
		socOptions(): SelectOption<number>[] {
			return Array.from(Array(20).keys()).map((i) => {
				const value = 5 + i * 5;
				return { value, name: this.fmtSocOption(value, this.rangePerSoc, distanceUnit()) };
			});
		},
	},
	methods: {
		formId(index: number, name: string): string {
			return `chargingplan-lp${this.id}-one-time-${index}-${name}`;
		},
		add(): void {
			const last = this.plans.at(-1);
			const date = last ? new Date(last.time) : new Date();
			date.setDate(date.getDate() + 1);
			const [hours, minutes] = DEFAULT_TARGET_TIME.split(":").map(Number);
			date.setHours(hours!, minutes!, 0, 0);
			this.$emit("updated", [
				...this.plans,
				{ time: date.toISOString(), soc: DEFAULT_TARGET_SOC },
			]);
		},
		update(index: number, field: "day" | "time" | "soc", event: Event): void {
			const row = { ...this.rows[index]!, [field]: (event.target as HTMLInputElement).value };
			const date = new Date(`${row.day}T${row.time}`);
			// incomplete or past input keeps the stored plan
			if (isNaN(date.getTime()) || date < new Date()) return;
			const plans = [...this.plans];
			plans[index] = { time: date.toISOString(), soc: Number(row.soc) };
			this.$emit("updated", plans);
		},
		remove(index: number): void {
			this.$emit(
				"updated",
				this.plans.filter((_, i) => i !== index)
			);
		},
	},
});
</script>

<style scoped>
.add-plan {
	margin-left: -0.5rem;
}
.plan-id {
	width: 2.5rem;
	color: var(--evcc-gray);
}
</style>
