<template>
	<div class="row">
		<div class="col-6 col-sm-4 mb-2 mb-sm-0">
			<select
				:id="`targetTimeLabel${id}`"
				v-model="selectedDay"
				class="form-select me-2"
				data-testid="target-day"
			>
				<option v-for="opt in dayOptions()" :key="opt.value" :value="opt.value">
					{{ opt.name }}
				</option>
			</select>
		</div>
		<div class="col-4 col-sm-3 mb-2 mb-sm-0">
			<input
				v-model="selectedTime"
				type="time"
				class="form-control mx-0"
				:step="60 * 5"
				required
			/>
		</div>
		<div class="offset-6 offset-sm-0 col-4 col-sm-3">
			<select class="form-select mx-0">
				<option
					v-for="opt in [
						{ value: 5, name: '5 %' },
						{ value: 10, name: '10 %' },
						{ value: 15, name: '15 %' },
						{ value: 20, name: '20 %' },
						{ value: 25, name: '25 %' },
						{ value: 30, name: '30 %' },
						{ value: 35, name: '35 %' },
						{ value: 40, name: '40 %' },
						{ value: 45, name: '45 %' },
						{ value: 50, name: '50 %' },
						{ value: 55, name: '55 %' },
						{ value: 60, name: '60 %' },
						{ value: 65, name: '65 %' },
						{ value: 70, name: '70 %' },
						{ value: 75, name: '75 %' },
						{ value: 80, name: '80 %' },
						{ value: 85, name: '85 %' },
						{ value: 90, name: '90 %' },
						{ value: 95, name: '95 %' },
						{ value: 100, name: '100 %' },
					]"
					:key="opt.value"
					:value="opt.value"
				>
					{{ opt.name }}
				</option>
			</select>
		</div>
		<div class="col-2 d-flex justify-content-end">
			<button type="button" class="btn btn-sm btn-link text-muted">
				<shopicon-regular-trash></shopicon-regular-trash>
			</button>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/trash";

import formatter from "../mixins/formatter";

const LAST_TARGET_TIME_KEY = "last_target_time";

export default {
	name: "ChargingPlanSettingsEntry",
	mixins: [formatter],
	props: {
		id: String,
		soc: Number,
		time: String,
	},
	emits: ["target-time-updated", "target-time-removed"],
	data: function () {
		return {
			selectedDay: null,
			selectedTime: null,
		};
	},
	computed: {
		timeInThePast: function () {
			const now = new Date();
			return now >= this.selectedTargetTime;
		},
		selectedTargetTime: function () {
			return new Date(`${this.selectedDay}T${this.selectedTime || "00:00"}`);
		},
		targetEnergyFormatted: function () {
			return this.fmtKWh(this.targetEnergy * 1e3, true, true, 1);
		},
		socOptions: function () {
			// a list of entries from 0 to 100 with a step of 5
			return Array.from(Array(21).keys())
				.map((i) => i * 5)
				.map((soc) => {
					return { value: soc, name: soc === 0 ? "--" : `${soc}%` };
				});
		},
	},
	watch: {
		time() {
			this.initInputFields();
		},
	},
	mounted() {
		this.initInputFields();
	},
	methods: {
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
		setTargetTime: function () {
			try {
				const hours = this.selectedTargetTime.getHours();
				const minutes = this.selectedTargetTime.getMinutes();
				window.localStorage[LAST_TARGET_TIME_KEY] = `${hours}:${minutes}`;
			} catch (e) {
				console.warn(e);
			}
			this.$emit("target-time-updated", this.selectedTargetTime);
		},
		removeTargetTime: function () {
			this.$emit("target-time-removed");
		},
	},
};
</script>

<style scoped>
@media (min-width: 992px) {
	.date-selection {
		width: 370px;
	}
}
.time-selection {
	flex-basis: 200px;
}
</style>
