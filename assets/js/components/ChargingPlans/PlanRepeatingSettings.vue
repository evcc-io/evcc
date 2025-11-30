<template>
	<div>
		<h5
			class="d-flex gap-3 align-items-baseline d-lg-none mb-4 fw-normal evcc-gray"
			data-testid="repeating-plan-title"
		>
			<span class="text-uppercase fs-6">
				{{ `${$t("main.chargingPlan.planNumber", { number: `#${number}` })}` }}
			</span>
			<small>
				{{ $t("main.chargingPlan.repeating") }}
			</small>
		</h5>

		<div v-if="showHeader" class="row d-none d-lg-flex mb-2">
			<div class="plan-id d-none d-lg-flex"></div>
			<div class="col-3">
				<label :for="formId('weekdays')">
					{{ $t("main.chargingPlan.weekdays") }}
				</label>
			</div>
			<div class="col-2">
				<label :for="formId('time')">
					{{ $t("main.chargingPlan.time") }}
				</label>
			</div>
			<div :class="showPrecondition ? 'col-3' : 'col-4'">
				<label :for="formId('goal')">
					{{ $t("main.chargingPlan.goal") }}
				</label>
			</div>
			<div v-if="showPrecondition" class="col-1">
				<label :for="formId('precondition')">
					{{ $t("main.chargingPlan.preconditionShort") }}
				</label>
			</div>
			<div class="col-1">
				<label :for="formId('active')"> {{ $t("main.chargingPlan.active") }} </label>
			</div>
		</div>
		<div class="row">
			<div class="plan-id d-none d-lg-flex align-items-center justify-content-start fs-6">
				#{{ number }}
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('weekdays')">
					{{ $t("main.chargingPlan.weekdays") }}
				</label>
			</div>
			<div class="col-7 col-lg-3 mb-2 mb-lg-0">
				<MultiSelect
					:id="formId('weekdays')"
					:value="selectedWeekdays"
					:options="dayOptions"
					:selectAllLabel="$t('main.chargingPlan.selectAll')"
					data-testid="repeating-plan-weekdays"
					@update:model-value="changeSelectedWeekdays"
				>
					{{ weekdaysLabel }}
				</MultiSelect>
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('time')">
					{{ $t("main.chargingPlan.time") }}
				</label>
			</div>
			<div class="col-7 col-lg-2 mb-2 mb-lg-0">
				<input
					:id="formId('time')"
					v-model="selectedTime"
					type="time"
					class="form-control mx-0 text-start"
					:step="60 * 5"
					data-testid="repeating-plan-time"
					required
					@change="update()"
				/>
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('goal')">
					{{ $t("main.chargingPlan.goal") }}
				</label>
			</div>
			<div :class="['col-7', showPrecondition ? 'col-lg-3' : 'col-lg-4', 'mb-2', 'mb-lg-0']">
				<select
					:id="formId('goal')"
					v-model="selectedSoc"
					class="form-select mx-0"
					data-testid="repeating-plan-soc"
					@change="update()"
				>
					<option v-for="opt in socOptions" :key="opt.value" :value="opt.value">
						{{ opt.name }}
					</option>
				</select>
			</div>
			<div v-if="showPrecondition" class="col-5 d-lg-none col-form-label">
				<label :for="formId('precondition')">
					{{ $t("main.chargingPlan.preconditionLong") }}
				</label>
			</div>
			<div
				v-if="showPrecondition"
				class="col-7 col-lg-1 mb-2 mb-lg-0 d-flex align-items-center"
			>
				<PreconditionSelect
					:id="formId('precondition')"
					v-model="selectedPrecondition"
					testid="repeating-plan-precondition"
				/>
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('active')">
					{{ $t("main.chargingPlan.active") }}
				</label>
			</div>
			<div class="col-3 col-lg-1 d-flex align-items-center">
				<div class="form-check form-switch">
					<input
						:id="formId('active')"
						v-model="selectedActive"
						class="form-check-input"
						type="checkbox"
						role="switch"
						data-testid="repeating-plan-active"
						:checked="selectedActive"
						tabindex="0"
						@change="update(true)"
					/>
				</div>
			</div>
			<div
				class="col-4 col-lg-1 d-flex align-items-center justify-content-end justify-content-lg-start"
			>
				<button
					v-if="showApply"
					type="button"
					class="btn btn-sm btn-outline-primary border-0 text-decoration-underline text-truncate"
					data-testid="repeating-plan-apply"
					tabindex="0"
					@click="update(true)"
				>
					<span class="d-lg-none">{{ $t("main.chargingPlan.update") }}</span>
					<shopicon-regular-checkmark
						size="s"
						class="flex-shrink-0 d-none d-lg-block"
					></shopicon-regular-checkmark>
				</button>
				<button
					v-else
					type="button"
					class="btn btn-sm btn-outline-secondary border-0"
					aria-label="Remove"
					tabindex="0"
					@click="$emit('removed', id())"
				>
					<shopicon-regular-trash size="s" class="flex-shrink-0"></shopicon-regular-trash>
				</button>
			</div>
		</div>
		<!-- Large screen precondition description -->
		<div class="plan-id-inset">
			<PreconditionSelect
				:id="formId('precondition')"
				v-model="selectedPrecondition"
				testid="repeating-plan-precondition"
				description-lg-only
			/>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/trash";
import { distanceUnit } from "@/units";
import MultiSelect from "../Helper/MultiSelect.vue";
import formatter from "@/mixins/formatter";
import deepEqual from "@/utils/deepEqual";
import PreconditionSelect from "./PreconditionSelect.vue";
import type { SelectOption } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "ChargingPlanRepeatingSettings",
	components: { MultiSelect, PreconditionSelect },
	mixins: [formatter],
	props: {
		number: Number,
		weekdays: { type: Array as PropType<number[]>, default: () => [] },
		time: String,
		tz: String,
		soc: Number,
		precondition: Number,
		showHeader: Boolean,
		active: Boolean,
		rangePerSoc: Number,
		formIdPrefix: String,
		showPrecondition: Boolean,
	},
	emits: ["updated", "removed"],
	data() {
		return {
			selectedWeekdays: this.weekdays,
			selectedTime: this.time,
			selectedSoc: this.soc,
			selectedActive: this.active,
			selectedPrecondition: this.precondition,
		};
	},
	computed: {
		dataChanged(): boolean {
			return (
				!deepEqual(this.weekdays, this.selectedWeekdays) ||
				this.time !== this.selectedTime ||
				this.soc !== this.selectedSoc ||
				this.active !== this.selectedActive ||
				this.precondition !== this.selectedPrecondition
			);
		},
		showApply(): boolean {
			return this.dataChanged && this.selectedActive;
		},
		weekdaysLabel(): string {
			return this.getShortenedWeekdaysLabel(this.selectedWeekdays);
		},
		socOptions(): SelectOption<number>[] {
			// a list of entries from 5 to 100 with a step of 5
			return Array.from(Array(20).keys())
				.map((i) => 5 + i * 5)
				.map(this.socOption);
		},
		dayOptions(): SelectOption<number>[] {
			return this.getWeekdaysList("long");
		},
	},
	watch: {
		weekdays(newValue: number[], oldValue: number[]) {
			if (!deepEqual(newValue, oldValue)) {
				this.selectedWeekdays = newValue;
			}
		},
		time(newValue: string) {
			this.selectedTime = newValue;
		},
		soc(newValue: number) {
			this.selectedSoc = newValue;
		},
		active(newValue: boolean) {
			this.selectedActive = newValue;
		},
		precondition(newValue: number) {
			this.selectedPrecondition = newValue;
		},
	},
	methods: {
		id(): number {
			return this.number || 0;
		},
		changeSelectedWeekdays(weekdays: number[]): void {
			this.selectedWeekdays = weekdays;
			this.update();
		},
		formId(name: string): string {
			return `${this.formIdPrefix}-${this.number}-${name}`;
		},
		socOption(value: number): SelectOption<number> {
			const name = this.fmtSocOption(value, this.rangePerSoc, distanceUnit());
			return { value, name };
		},
		update(forceSave = false): void {
			const plan = {
				weekdays: this.selectedWeekdays,
				time: this.selectedTime,
				soc: this.selectedSoc,
				tz: this.tz,
				active: this.selectedActive,
				precondition: this.selectedPrecondition,
			};

			if (forceSave || !this.selectedActive) {
				this.$emit("updated", plan);
			}
		},
	},
});
</script>
<style scoped>
.plan-id-inset {
	margin-left: 2.5rem;
}
.plan-id {
	width: 2.5rem;
	color: var(--evcc-gray);
}
</style>
