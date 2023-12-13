<template>
	<div>
		<div class="row d-none d-lg-flex mb-2">
			<div class="col-6 col-lg-4">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.day") }}
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
			<div class="col-6 d-lg-none col-form-label">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.day") }}
				</label>
			</div>
			<div class="col-6 col-lg-4 mb-2 mb-lg-0">
				<select
					:id="formId('day')"
					v-model="selectedDay"
					class="form-select me-2"
					data-testid="plan-day"
				>
					<option v-for="opt in dayOptions()" :key="opt.value" :value="opt.value">
						{{ opt.name }}
					</option>
				</select>
			</div>
			<div class="col-6 d-lg-none col-form-label">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.time") }}
				</label>
			</div>
			<div class="col-6 col-lg-2 mb-2 mb-lg-0">
				<input
					:id="formId('time')"
					v-model="selectedTime"
					type="time"
					class="form-control mx-0"
					:step="60 * 5"
					data-testid="plan-time"
					required
				/>
			</div>
			<div class="col-6 d-lg-none col-form-label">
				<label :for="formId('goal')">
					{{ $t("main.chargingPlan.goal") }}
				</label>
			</div>
			<div class="col-6 col-lg-3 mb-2 mb-lg-0">
				<select
					v-if="socBasedPlanning"
					:id="formId('goal')"
					v-model="selectedSoc"
					class="form-select mx-0"
					data-testid="plan-soc"
				>
					<option v-for="opt in socOptions" :key="opt.value" :value="opt.value">
						{{ opt.name }}
					</option>
				</select>
				<select
					v-else
					:id="formId('goal')"
					v-model="selectedEnergy"
					class="form-select mx-0"
					data-testid="plan-energy"
				>
					<option v-for="opt in energyOptions" :key="opt.energy" :value="opt.energy">
						{{ opt.text }}
					</option>
				</select>
			</div>
			<div class="col-6 d-lg-none col-form-label">
				<label :for="formId('active')">
					{{ $t("main.chargingPlan.active") }}
				</label>
			</div>
			<div class="col-2 d-flex align-items-center justify-content-start">
				<div class="form-check form-switch">
					<input
						:id="formId('active')"
						class="form-check-input"
						type="checkbox"
						role="switch"
						data-testid="plan-active"
						:checked="!isNew"
						:disabled="timeInThePast"
						@change="toggle"
					/>
				</div>
				<button
					v-if="dataChanged && !isNew"
					type="button"
					class="btn btn-sm btn-outline-primary ms-3 border-0 text-decoration-underline"
					data-testid="plan-apply"
					:disabled="timeInThePast"
					@click="update"
				>
					{{ $t("main.chargingPlan.update") }}
				</button>
			</div>
		</div>
		<p class="mb-0">
			<span v-if="timeInThePast" class="d-block text-danger my-2">
				{{ $t("main.targetCharge.targetIsInThePast") }}
			</span>
		</p>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/checkmark";
import { distanceUnit } from "../units";

import formatter from "../mixins/formatter";
import { energyOptions } from "../utils/energyOptions";

const LAST_TARGET_TIME_KEY = "last_target_time";
const DEFAULT_TARGET_TIME = "7:00";

export default {
	name: "ChargingPlanSettingsEntry",
	mixins: [formatter],
	props: {
		id: String,
		soc: Number,
		energy: Number,
		time: String,
		rangePerSoc: Number,
		socPerKwh: Number,
		vehicleCapacity: Number,
		socBasedPlanning: Boolean,
	},
	emits: ["plan-updated", "plan-removed"],
	data: function () {
		return {
			selectedDay: null,
			selectedTime: null,
			selectedSoc: this.soc,
			selectedEnergy: this.energy,
			enabled: false,
		};
	},
	computed: {
		selectedDate: function () {
			return new Date(`${this.selectedDay}T${this.selectedTime || "00:00"}`);
		},
		socOptions: function () {
			// a list of entries from 5 to 100 with a step of 5
			return Array.from(Array(20).keys())
				.map((i) => 5 + i * 5)
				.map(this.socOption);
		},
		energyOptions: function () {
			const options = energyOptions(
				0,
				this.vehicleCapacity || 100,
				this.socPerKwh,
				this.fmtKWh,
				"-"
			);
			// remove the first entry (0)
			return options.slice(1);
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
		timeInThePast: function () {
			const now = new Date();
			return now >= this.selectedDate;
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
		initInputFields: function () {
			if (!this.selectedSoc) {
				this.selectedSoc = 100;
			}
			if (!this.selectedEnergy) {
				this.selectedEnergy = this.vehicleCapacity || 10;
			}

			let time = this.time;
			if (!time) {
				// no time but existing selection, keep it
				if (this.selectedDay && this.selectedTime) {
					return;
				}
				time = this.defaultTime();
			}
			const date = new Date(time);
			this.selectedDay = this.fmtDayString(date);
			this.selectedTime = this.fmtTimeString(date);
		},
		dayOptions: function () {
			const options = [];
			const date = new Date();
			const labels = [
				this.$t("main.targetCharge.today"),
				this.$t("main.targetCharge.tomorrow"),
			];
			for (let i = 0; i < 7; i++) {
				const dayNumber = date.toLocaleDateString(this.$i18n.locale, {
					month: "short",
					day: "numeric",
				});
				const dayName =
					labels[i] || date.toLocaleDateString(this.$i18n.locale, { weekday: "long" });
				options.push({
					value: this.fmtDayString(date),
					name: `${dayNumber} (${dayName})`,
				});
				date.setDate(date.getDate() + 1);
			}
			return options;
		},
		update: function () {
			try {
				const hours = this.selectedDate.getHours();
				const minutes = this.selectedDate.getMinutes();
				window.localStorage[LAST_TARGET_TIME_KEY] = `${hours}:${minutes}`;
			} catch (e) {
				console.warn(e);
			}
			this.$emit("plan-updated", {
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
				this.$emit("plan-removed");
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
