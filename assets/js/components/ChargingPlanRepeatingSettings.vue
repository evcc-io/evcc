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
			<div class="col-6 col-lg-3">
				<label :for="formId('weekdays')">
					{{ $t("main.chargingPlan.weekdays") }}
				</label>
			</div>
			<div class="col-6 col-lg-2">
				<label :for="formId('time')">
					{{ $t("main.chargingPlan.time") }}
				</label>
			</div>
			<div class="col-6 col-lg-3">
				<label :for="formId('goal')">
					{{ $t("main.chargingPlan.goal") }}
				</label>
			</div>
			<div class="col-6 col-lg-1">
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
			<div class="col-7 col-lg-3 mb-2 mb-lg-0">
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
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('active')">
					{{ $t("main.chargingPlan.active") }}
				</label>
			</div>
			<div class="col-2 col-lg-1 d-flex align-items-center">
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
			<div class="col-5 col-lg-2 d-flex align-items-center">
				<button
					v-if="showApply"
					type="button"
					class="btn btn-sm btn-outline-primary border-0 text-decoration-underline"
					data-testid="repeating-plan-apply"
					tabindex="0"
					@click="update(true)"
				>
					{{ $t("main.chargingPlan.update") }}
				</button>
				<button
					v-else
					type="button"
					class="btn btn-sm btn-outline-secondary border-0"
					aria-label="Remove"
					tabindex="0"
					@click="$emit('removed', id)"
				>
					<shopicon-regular-trash size="s" class="flex-shrink-0"></shopicon-regular-trash>
				</button>
			</div>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/trash";
import { distanceUnit } from "../units";
import MultiSelect from "./MultiSelect.vue";
import formatter from "../mixins/formatter";
import deepEqual from "../utils/deepEqual";
export default {
	name: "ChargingPlanRepeatingSettings",
	components: {
		MultiSelect,
	},
	mixins: [formatter],
	props: {
		number: Number,
		weekdays: { type: Array, default: () => [] },
		time: String,
		tz: String,
		soc: Number,
		showHeader: Boolean,
		active: Boolean,
		rangePerSoc: Number,
		formIdPrefix: String,
	},
	emits: ["updated", "removed"],
	data: function () {
		return {
			selectedWeekdays: this.weekdays,
			selectedTime: this.time,
			selectedSoc: this.soc,
			selectedActive: this.active,
		};
	},
	computed: {
		dataChanged: function () {
			return (
				!deepEqual(this.weekdays, this.selectedWeekdays) ||
				this.time !== this.selectedTime ||
				this.soc !== this.selectedSoc ||
				this.active !== this.selectedActive
			);
		},
		showApply: function () {
			return this.dataChanged && this.selectedActive;
		},
		weekdaysLabel: function () {
			return this.getShortenedWeekdaysLabel(this.selectedWeekdays);
		},
		socOptions: function () {
			// a list of entries from 5 to 100 with a step of 5
			return Array.from(Array(20).keys())
				.map((i) => 5 + i * 5)
				.map(this.socOption);
		},
		dayOptions: function () {
			return this.getWeekdaysList("long");
		},
	},
	watch: {
		weekdays(newValue, oldValue) {
			if (!deepEqual(newValue, oldValue)) {
				this.selectedWeekdays = newValue;
			}
		},
		time(newValue) {
			this.selectedTime = newValue;
		},
		soc(newValue) {
			this.selectedSoc = newValue;
		},
		active(newValue) {
			this.selectedActive = newValue;
		},
	},
	methods: {
		changeSelectedWeekdays: function (weekdays) {
			this.selectedWeekdays = weekdays;
			this.update();
		},
		formId: function (name) {
			return `${this.formIdPrefix}-${this.number}-${name}`;
		},
		socOption: function (value) {
			const name = this.fmtSocOption(value, this.rangePerSoc, distanceUnit());
			return { value, name };
		},
		update: function (forceSave = false) {
			const plan = {
				weekdays: this.selectedWeekdays,
				time: this.selectedTime,
				soc: this.selectedSoc,
				tz: this.tz,
				active: this.selectedActive,
			};

			if (forceSave || !this.selectedActive) {
				this.$emit("updated", plan);
			}
		},
	},
};
</script>
<style scoped>
.plan-id {
	width: 2.5rem;
	color: var(--evcc-gray);
}
</style>
