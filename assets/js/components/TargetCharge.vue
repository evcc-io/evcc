<template>
	<div>
		<LabelAndValue
			class="root flex-grow-1"
			:label="$t('main.targetCharge.title')"
			:class="disabled ? 'opacity-0' : 'opacity-1'"
		>
			<button
				class="btn btn-link p-0 value text-center"
				:class="targetChargeEnabled ? 'evcc-default-text' : 'text-gray'"
				data-bs-toggle="modal"
				:data-bs-target="`#${modalId}`"
				:disabled="disabled"
			>
				<strong v-if="targetChargeEnabled">{{ targetTimeLabel() }}</strong>
				<span v-else>{{ $t("main.targetCharge.setTargetTime") }}</span>
			</button>
		</LabelAndValue>

		<Teleport to="body">
			<div
				:id="modalId"
				class="modal fade text-dark"
				data-bs-backdrop="true"
				tabindex="-1"
				role="dialog"
				aria-hidden="true"
			>
				<div
					class="modal-dialog modal-dialog-centered modal-dialog-scrollable"
					role="document"
				>
					<div class="modal-content">
						<div class="modal-header">
							<h5 class="modal-title">
								{{ $t("main.targetCharge.modalTitle") }}
							</h5>
							<button
								type="button"
								class="btn-close"
								data-bs-dismiss="modal"
								aria-label="Close"
							></button>
						</div>
						<form @submit.prevent="setTargetTime">
							<div class="modal-body">
								<div class="form-group">
									<!-- eslint-disable vue/no-v-html -->
									<label for="targetTimeLabel" class="mb-3">
										{{ $t("main.targetCharge.description", { targetSoC }) }}
									</label>
									<!-- eslint-enable vue/no-v-html -->
									<div
										class="d-flex justify-content-between"
										:style="{ 'max-width': '350px' }"
									>
										<select
											v-model="selectedDay"
											class="form-select me-2"
											:style="{ 'flex-basis': '60%' }"
										>
											<option
												v-for="opt in dayOptions()"
												:key="opt.value"
												:value="opt.value"
											>
												{{ opt.name }}
											</option>
										</select>
										<input
											v-model="selectedTime"
											type="time"
											class="form-control ms-2"
											:style="{ 'flex-basis': '40%' }"
											:step="60 * 5"
											required
										/>
									</div>
								</div>
								<p v-if="!selectedTargetTimeValid" class="text-danger mb-0">
									{{ $t("main.targetCharge.targetIsInThePast") }}
								</p>
								<p class="small mt-3 text-muted mb-0">
									<strong class="text-evcc">
										{{ $t("main.targetCharge.experimentalLabel") }}:
									</strong>
									{{ $t("main.targetCharge.experimentalText") }}
									<a
										href="https://github.com/evcc-io/evcc/discussions/1433"
										target="_blank"
										>GitHub Discussions</a
									>.
								</p>
							</div>
							<div class="modal-footer d-flex justify-content-between">
								<button
									type="button"
									class="btn btn-outline-secondary"
									data-bs-dismiss="modal"
									@click="removeTargetTime"
								>
									{{ $t("main.targetCharge.remove") }}
								</button>
								<button
									type="submit"
									class="btn btn-primary"
									data-bs-dismiss="modal"
									:disabled="!selectedTargetTimeValid"
								>
									{{ $t("main.targetCharge.activate") }}
								</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		</Teleport>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/plus";
import "@h2d2/shopicons/es/filled/edit";
import LabelAndValue from "./LabelAndValue.vue";

import formatter from "../mixins/formatter";

const DEFAULT_TARGET_TIME = "7:00";
const LAST_TARGET_TIME_KEY = "last_target_time";

export default {
	name: "TargetCharge",
	components: { LabelAndValue },
	mixins: [formatter],
	props: {
		id: [String, Number],
		targetTime: String,
		targetTimeActive: Boolean,
		targetSoC: Number,
		disabled: Boolean,
	},
	emits: ["target-time-updated", "target-time-removed"],
	data: function () {
		return { selectedDay: null, selectedTime: null };
	},
	computed: {
		targetChargeEnabled: function () {
			return this.targetTime;
		},
		selectedTargetTimeValid: function () {
			const now = new Date();
			return now < this.selectedTargetTime;
		},
		selectedTargetTime: function () {
			return new Date(`${this.selectedDay}T${this.selectedTime || "00:00"}`);
		},
		modalId: function () {
			return `targetChargeModal_${this.id}`;
		},
	},
	watch: {
		targetTimeLabel: function () {
			const targetDate = new Date(this.targetTime);
			return this.$t("main.targetCharge.activeLabel", {
				time: this.fmtAbsoluteDate(targetDate),
			});
		},
		targetTime() {
			this.initInputFields();
		},
	},
	mounted: function () {
		this.initInputFields();
	},
	methods: {
		// not computed because it needs to update over time
		targetTimeLabel: function () {
			if (this.targetChargeEnabled) {
				const targetDate = new Date(this.targetTime);
				return this.$t("main.targetCharge.activeLabel", {
					time: this.fmtAbsoluteDate(targetDate),
				});
			}
			return this.$t("main.targetCharge.inactiveLabel");
		},
		defaultDate: function () {
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
		initInputFields: function () {
			let date = this.defaultDate();
			let targetTimeInTheFuture = new Date(this.targetTime) > new Date();
			if (this.targetChargeEnabled && targetTimeInTheFuture) {
				date = new Date(this.targetTime);
			}
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
				const dayNumber = date.toLocaleDateString("default", {
					month: "short",
					day: "numeric",
				});
				const dayName =
					labels[i] || date.toLocaleDateString("default", { weekday: "long" });
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
.value {
	font-size: 18px;
	line-height: 1.2;
	border: none;
}
.root {
	transition: opacity var(--evcc-transition-medium) linear;
}
.value:hover {
	color: var(--bs-color-white);
}
</style>
