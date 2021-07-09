<template>
	<div class="d-flex justify-content-between align-items-center">
		<small class="text-secondary">
			<span v-if="minSoCActive">
				<fa-icon class="text-muted me-1" icon="exclamation-circle"></fa-icon>
				{{ $t("main.vehicleSubline.mincharge", { soc: minSoC }) }}
			</span>
		</small>
		<button
			class="target-time-button btn btn-link btn-sm pe-0"
			:class="{
				invisible: !targetSoC,
				'text-primary': timerActive,
				'text-secondary': !timerActive,
			}"
			data-bs-toggle="modal"
			data-bs-target="#targetChargeModal"
		>
			{{ targetTimeLabel() }}<fa-icon class="ms-1" icon="clock"></fa-icon>
		</button>

		<div
			id="targetChargeModal"
			class="modal fade"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">Zielzeit festlegen</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<form @submit.prevent="saveTargetTime">
						<div class="modal-body">
							<div class="form-group">
								<label for="targetTimeLabel" class="mb-3">
									Wann soll das Fahrzeug auf
									<strong>{{ targetSoC }}%</strong> geladen sein?
								</label>
								<div
									class="d-flex justify-content-between"
									:style="{ 'max-width': '350px' }"
								>
									<select
										class="form-select me-2"
										v-model="selectedDay"
										:style="{ 'flex-basis': '60%' }"
									>
										<option
											v-for="opt in dayOptions()"
											:value="opt.value"
											:key="opt.value"
										>
											{{ opt.name }}
										</option>
									</select>
									<input
										type="time"
										class="form-control ms-2"
										:style="{ 'flex-basis': '40%' }"
										v-model="selectedTime"
										:step="60 * 5"
										required
									/>
								</div>
							</div>
							<p v-if="selectedTargetTimeValid"></p>
							<p class="text-danger" v-if="!selectedTargetTimeValid">
								Zeitpunkt liegt in der Vergangenheit.
							</p>
						</div>
						<div class="modal-footer d-flex justify-content-between">
							<button
								type="button"
								class="btn btn-outline-secondary"
								@click="removeTargetTime"
							>
								Keine Zeilzeit
							</button>
							<button
								type="submit"
								class="btn btn-success"
								:disabled="!selectedTargetTimeValid"
							>
								Zielzeit aktivieren
							</button>
						</div>
					</form>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "VehicleSubline",
	props: {
		socCharge: Number,
		minSoC: Number,
		timerActive: Boolean,
		timerSet: Boolean,
		targetTime: String,
		targetSoC: Number,
	},
	computed: {
		minSoCActive: function () {
			return this.minSoC > 0 && this.socCharge < this.minSoC;
		},
		targetChargeEnabled: function () {
			return this.targetTime && this.timerSet;
		},
		selectedTargetTimeValid: function () {
			const now = new Date();
			return now < this.selectedTargetTime;
		},
		selectedTargetTime: function () {
			return new Date(`${this.selectedDay}T${this.selectedTime || "00:00"}`);
		},
	},
	data: function () {
		return { selectedDay: null, selectedTime: null };
	},
	mounted: function () {
		this.initInputFields();
	},
	watch: {
		targetTime() {
			this.initInputFields();
		},
	},
	methods: {
		// not computed because it needs to update over time
		targetTimeLabel: function () {
			if (this.targetChargeEnabled) {
				const targetDate = new Date(this.targetTime);
				return `bis ${this.fmtAbsoluteDate(targetDate)} Uhr`;
			}
			return "Zielzeit";
		},
		defaultDate: function () {
			const now = new Date();
			// 12 hrs from now
			now.setHours(now.getHours() + 12);
			// round to quarter hour
			now.setMinutes(Math.ceil(now.getMinutes() / 15) * 15);
			return now;
		},
		initInputFields: function () {
			const date = this.targetChargeEnabled ? new Date(this.targetTime) : this.defaultDate();
			this.selectedDay = this.fmtDayString(date);
			this.selectedTime = this.fmtTimeString(date);
		},
		dayOptions: function () {
			const options = [];
			const date = new Date();
			const labels = ["heute", "morgen"];
			for (let i = 0; i < 7; i++) {
				const dayNumber = date.toLocaleDateString("default", {
					month: "long",
					day: "numeric",
				});
				const dayName =
					labels[i] || date.toLocaleDateString("default", { weekday: "long" });
				options.push({
					value: date.toISOString().split("T")[0],
					name: `${dayNumber} (${dayName})`,
				});
				date.setDate(date.getDate() + 1);
			}
			return options;
		},
		minTime: function () {
			return new Date().toISOString().split("T")[1].slice(0, -8);
		},
		removeTargetTime: function () {
			this.$emit("target-time-updated", new Date(null));
			this.closeModal();
		},
		saveTargetTime: function () {
			this.$emit("target-time-updated", this.selectedTargetTime);
			this.closeModal();
		},
	},
	mixins: [formatter],
};
</script>
<style scoped>
.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.25s ease-in;
}
.fade-enter,
.fade-leave-to {
	opacity: 0;
}
.dialog {
	position: fixed;
	top: 0;
	left: 0;
	z-index: 1050;
	width: 100%;
	height: 100%;
	background-color: rgba(0, 0, 0, 0.5);
	overflow: hidden;
	outline: 0;
}
</style>
