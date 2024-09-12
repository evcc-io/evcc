<template>
	<div>
		{{ weekdaysValues }}
		<div class="row d-none d-lg-flex mb-2">
			<div class="col-6 col-lg-4">
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
			<div class="col-6 col-lg-1" />
		</div>
		<div class="row">
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.weekdays") }}
				</label>
			</div>
			<div class="col-7 col-lg-4 mb-2 mb-lg-0">
				<MultiSelect
					id="chargingPlanWeekdaySelect"
					:options="dayOptions()"
					:selectAllLabel="$t('main.chargingPlan.selectAll')"
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
					data-testid="plan-time"
					required
					@change="preview"
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
					data-testid="plan-soc"
					@change="preview"
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
						class="form-check-input"
						type="checkbox"
						role="switch"
						data-testid="plan-active"
						:checked="isActive"
						@change="toggle"
					/>
				</div>
			</div>
			<div class="col-1 mx-auto d-flex align-items-center justify-content-start">
				<button
					type="button"
					class="btn btn-sm btn-outline-secondary border-0"
					data-testid="plan-delete"
					@click="update"
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

const WEEKDAYS = ["monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"];

export default {
	name: "ChargingPlanRepetitiveSettingsEntry",
	components: {
		MultiSelect,
	},
	mixins: [formatter],
	props: {
		id: String,
		weekdays: Array,
		time: String,
		soc: Number,
		active: Boolean,
		socBasedPlanning: Boolean,
	},
	emits: ["static-plan-updated", "static-plan-removed", "plan-preview"],
	data: function () {
		return {
			selectedWeekdays: this.weekdays,
			selectedTime: this.time,
			selectedSoc: this.soc,
			isActive: this.active,
		};
	},
	computed: {
		weekdaysLabel: function () {
			let label = "";
			this.selectedWeekdays.sort(function (a, b) {
				return a - b;
			});

			for (let index = 0; index < this.selectedWeekdays.length; index++) {
				label +=
					this.$t(`main.chargingPlan.${WEEKDAYS[this.selectedWeekdays[index]]}`).slice(
						0,
						2
					) + ", ";
			}

			return label;
		},
		selectedDate: function () {
			return new Date(`${this.selectedDay}T${this.selectedTime || "00:00"}`);
		},
		socOptions: function () {
			// a list of entries from 5 to 100 with a step of 5
			return Array.from(Array(20).keys())
				.map((i) => 5 + i * 5)
				.map(this.socOption);
		},
		originalData: function () {
			if (this.isNew) {
				return {};
			}
			return {
				soc: this.soc,
				energy: this.energy,
				day: this.fmtDayString(new Date(this.time)),
				time: this.fmtTimeString(new Date(this.time)),
			};
		},
		dataChanged: function () {
			const dateChanged =
				this.originalData.day != this.selectedDay ||
				this.originalData.time != this.selectedTime;
			const goalChanged = this.socBasedPlanning
				? this.originalData.soc != this.selectedSoc
				: this.originalData.energy != this.selectedEnergy;
			return dateChanged || goalChanged;
		},
		isNew: function () {
			return !this.time && (!this.soc || !this.energy);
		},
	},
	watch: {
		time() {
			this.initInputFields();
		},
		soc() {
			if (this.soc) {
				this.selectedSoc = this.soc;
			}
		},
		energy() {
			if (this.energy) {
				this.selectedEnergy = this.energy;
			}
		},
	},
	mounted() {
		this.initInputFields();
	},
	methods: {
		formId: function (name) {
			return `chargingplan-${this.id}-${name}`;
		},
		socOption: function (value) {
			const name = this.fmtSocOption(value, this.rangePerSoc, distanceUnit());
			return { value, name };
		},
		dayOptions: function () {
			return WEEKDAYS.map((weekday, index) => {
				return {
					value: index,
					name: this.$t(`main.chargingPlan.${weekday}`),
				};
			});
		},
		update: function () {
			try {
				const hours = this.selectedDate.getHours();
				const minutes = this.selectedDate.getMinutes();
				window.localStorage[LAST_TARGET_TIME_KEY] = `${hours}:${minutes}`;
				if (this.selectedSoc) {
					window.localStorage[LAST_SOC_GOAL_KEY] = this.selectedSoc;
				}
				if (this.selectedEnergy) {
					window.localStorage[LAST_ENERGY_GOAL_KEY] = this.selectedEnergy;
				}
			} catch (e) {
				console.warn(e);
			}
			this.$emit("static-plan-updated", {
				time: this.selectedDate,
				soc: this.selectedSoc,
				energy: this.selectedEnergy,
			});
		},
		preview: function () {
			this.$emit("plan-preview", {
				time: this.selectedDate,
				soc: this.selectedSoc,
				energy: this.selectedEnergy,
			});
		},
		toggle: function (e) {
			const { checked } = e.target;
			if (checked) {
				this.update();
			} else {
				this.$emit("static-plan-removed");
			}
			this.enabled = checked;
		},
		defaultTime: function () {
			const [hours, minutes] = (
				window.localStorage[LAST_TARGET_TIME_KEY] || DEFAULT_TARGET_TIME
			).split(":");

			const target = new Date();
			target.setSeconds(0);
			target.setMinutes(minutes);
			target.setHours(hours);
			// today or tomorrow?
			const isInPast = target < new Date();
			if (isInPast) {
				target.setDate(target.getDate() + 1);
			}
			return target;
		},
	},
};
</script>
