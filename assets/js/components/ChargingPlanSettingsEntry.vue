<template>
	<div>
		<div class="row d-none d-lg-flex mb-2">
			<div class="col-6 col-lg-4">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.day") }}
				</label>
			</div>
			<div class="col-6 col-lg-3">
				<label :for="formId('time')">
					{{ $t("main.chargingPlan.time") }}
				</label>
			</div>
			<div class="col-6 col-lg-3">
				<label :for="formId('goal')">
					{{ $t("main.chargingPlan.goal") }}
				</label>
			</div>
			<div class="col-2"></div>
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
			<div class="col-6 col-lg-3 mb-2 mb-lg-0">
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
			<div class="col-12 col-lg-2 d-flex justify-content-end align-items-baseline">
				<button
					type="button"
					class="btn evcc-default-text text-decoration-underline"
					@click="removePlan"
				>
					{{ $t("main.chargingPlan.remove") }}
				</button>
			</div>
		</div>
	</div>
</template>

<script>
import { distanceUnit } from "../units";

import formatter from "../mixins/formatter";
import { energyOptions } from "../utils/energyOptions";

const LAST_TARGET_TIME_KEY = "last_target_time";

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
		};
	},
	computed: {
		timeInThePast: function () {
			const now = new Date();
			return now >= this.selectedDate;
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
	},
	watch: {
		time() {
			this.initInputFields();
		},
		selectedDate() {
			this.updatePlan();
		},
		selectedSoc() {
			this.updatePlan();
		},
		selectedEnergy() {
			this.updatePlan();
		},
		soc() {
			this.selectedSoc = this.soc;
		},
		energy() {
			this.selectedEnergy = this.energy;
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
			const date = new Date(this.time);
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
		updatePlan: function () {
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
		removePlan: function () {
			this.$emit("plan-removed");
		},
	},
};
</script>
