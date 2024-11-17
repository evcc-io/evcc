<template>
	<div>
		<div class="row d-none d-lg-flex mb-2">
			<div class="col-id d-none d-lg-flex"></div>
			<div class="col-6 col-lg-3">
				<label :for="formId('day')">
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
			<div class="col-id d-none d-lg-flex align-items-center justify-content-start">
				<h5>#{{ id + 2 }}</h5>
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
					@update:model-value="changeSelectedWeekdays"
				>
					{{ weekdaysLabel }}
				</MultiSelect>
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.time") }}
				</label>
			</div>
			<div class="col-7 col-lg-2 mb-2 mb-lg-0">
				<input
					:id="formId('time')"
					v-model="selectedTime"
					type="time"
					class="form-control mx-0"
					:step="60 * 5"
					data-testid="repeating-plan-time"
					required
					@change="update"
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
					@change="update"
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
			<div class="col-1 d-flex align-items-center justify-content-start">
				<div class="form-check form-switch">
					<input
						:id="formId('active')"
						v-model="selectedActive"
						class="form-check-input"
						type="checkbox"
						role="switch"
						data-testid="repeating-plan-active"
						:checked="selectedActive"
						@change="update"
					/>
				</div>
			</div>
			<div class="col-1 mx-auto d-flex align-items-center justify-content-start">
				<button
					type="button"
					class="btn btn-sm btn-outline-secondary border-0"
					data-testid="plan-delete"
					@click="$emit('repeating-plan-removed', id)"
				>
					<shopicon-regular-trash size="s" class="flex-shrink-0"></shopicon-regular-trash>
				</button>
			</div>
			<div class="col-1"></div>
			<!-- Adds space to the right side which is needed due to the extra column containing the trash-icons -->
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/trash";
import { distanceUnit } from "../units";
import MultiSelect from "./MultiSelect.vue";
import formatter from "../mixins/formatter";

export default {
	name: "ChargingPlanRepeatingSettings",
	components: {
		MultiSelect,
	},
	mixins: [formatter],
	props: {
		id: Number,
		weekdays: { type: Array, default: () => [] },
		time: String,
		soc: Number,
		active: Boolean,
		rangePerSoc: Number,
		formIdPrefix: String,
	},
	emits: ["repeating-plan-updated", "repeating-plan-removed"],
	data: function () {
		return {
			selectedWeekdays: this.weekdays,
			selectedTime: this.time,
			selectedSoc: this.soc,
			selectedActive: this.active,
		};
	},
	computed: {
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
		weekdays(newValue) {
			this.selectedWeekdays = newValue;
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
			return `${this.formIdPrefix}-${this.id}-${name}`;
		},
		socOption: function (value) {
			const name = this.fmtSocOption(value, this.rangePerSoc, distanceUnit());
			return { value, name };
		},
		update: function () {
			this.$emit("repeating-plan-updated", {
				id: this.id,
				weekdays: this.selectedWeekdays,
				time: this.selectedTime,
				soc: this.selectedSoc,
				active: this.selectedActive,
			});
		},
	},
};
</script>
<style scoped>
.col-id {
	width: 4%;
	padding-right: 0;
	padding-left: 0;
	color: var(--evcc-gray);
}
</style>
