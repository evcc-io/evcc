<template>
	<div class="subline d-flex justify-content-between align-items-center">
		<div class="min-soc-status text-secondary">
			<div v-if="minSoCActive">
				<fa-icon class="text-muted mr-1" icon="exclamation-circle"></fa-icon>
				Mindestladung bis {{ minSoC }}%
			</div>
		</div>
		<button
			v-if="targetSoC"
			class="target-time-button btn btn-link btn-sm pr-0 text-secondary"
			:class="{ 'text-dark': timerSet, 'text-muted': !timerSet }"
			@click="selectTargetTime"
		>
			{{ targetTimeLabel() }}<fa-icon class="ml-1" icon="clock"></fa-icon>
		</button>

		<transition name="fade">
			<div class="dialog" tabindex="-1" role="dialog" v-if="targetTimeModalActive">
				<div class="modal-dialog modal-dialog-centered" role="document">
					<div class="modal-content">
						<div class="modal-header">
							<h4 class="modal-title font-weight-bold">Zielzeit festlegen</h4>
							<button type="button" class="close" @click="closeModal">
								<span aria-hidden="true">&times;</span>
							</button>
						</div>
						<div class="modal-body">
							<form>
								<div class="form-group">
									<label for="targetTimeLabel"
										>Wann soll das Fahrzeug auf
										<strong>{{ targetSoC }}%</strong> geladen sein?</label
									>
									<div class="d-flex">
										<select
											class="form-control mr-3"
											:style="{ 'flex-basis': '66%' }"
											v-model="selectedDay"
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
											class="form-control"
											v-model="selectedTime"
											:style="{ 'flex-basis': '33%' }"
											:step="60 * 5"
											required
										/>
									</div>
								</div>
							</form>
							<p v-if="selectedTargetTimeValid"></p>
							<p class="text-danger" v-if="!selectedTargetTimeValid">
								Zeitpunkt liegt in der Vergangenheit.
							</p>
						</div>
						<div class="modal-footer d-flex justify-content-between">
							<button
								type="button"
								class="btn btn-outline-secondary"
								@click="saveTargetTime"
							>
								Keine Zeilzeit
							</button>
							<button
								type="button"
								class="btn btn-primary"
								@click="saveTargetTime"
								:disabled="!selectedTargetTimeValid"
							>
								Zielzeit aktivieren
							</button>
						</div>
					</div>
				</div>
			</div>
		</transition>
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
		return { targetTimeModalActive: false, selectedDay: null, selectedTime: null };
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
				if (this.timerActive) {
					return `Lädt ${this.fmtRelativeTime(targetDate)} bis ${this.targetSoC}%`;
				} else {
					return `Geplant bis ${this.fmtAbsoluteDate(targetDate)} bis ${this.targetSoC}%`;
				}
			}
			return "Zielzeit festlegen";
		},
		minDate: function () {
			return new Date().toISOString().split("T")[0];
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
			const date = this.targetTime ? new Date(this.targetTime) : this.defaultDate();

			this.selectedDay = date.toISOString().split("T")[0];
			this.selectedTime = date.toTimeString().slice(0, 5);
		},
		dayOptions: function () {
			const options = [];
			const date = new Date();
			const labels = ["heute", "morgen"];
			for (let i = 0; i < 7; i++) {
				const dayNumber = date.toLocaleDateString("de-DE", {
					month: "long",
					day: "numeric",
				});
				const dayName = labels[i] || date.toLocaleDateString("de-DE", { weekday: "long" });
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
		selectTargetTime: function () {
			this.targetTimeModalActive = true;
		},
		closeModal: function () {
			this.targetTimeModalActive = false;
		},
		saveTargetTime: function () {
			alert("gespeichert");
			this.closeModal();
		},
	},
	mixins: [formatter],
};
</script>
<style scoped>
.min-soc-status,
.target-time-button {
	font-size: 0.875rem;
}
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
