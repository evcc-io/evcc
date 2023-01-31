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
				:disabled="disabled"
				@click="openModal"
			>
				<strong v-if="targetChargeEnabled">{{ targetTimeLabel() }}</strong>
				<span v-else>{{ $t("main.targetCharge.setTargetTime") }}</span>
			</button>
		</LabelAndValue>

		<Teleport to="body">
			<div
				:id="modalId"
				ref="modal"
				class="modal fade text-dark modal-xl"
				data-bs-backdrop="true"
				tabindex="-1"
				role="dialog"
				aria-hidden="true"
			>
				<div class="modal-dialog modal-dialog-centered" role="document">
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
								<div
									class="form-group d-lg-flex align-items-baseline mb-2 justify-content-between"
								>
									<!-- eslint-disable vue/no-v-html -->
									<label for="targetTimeLabel" class="mb-3 me-3">
										<span v-if="socBasedCharging">
											{{
												$t("main.targetCharge.descriptionSoc", {
													targetSoc,
												})
											}}
										</span>
										<span v-else>
											{{
												$t("main.targetCharge.descriptionEnergy", {
													targetEnergy: targetEnergyFormatted,
												})
											}}
										</span>
									</label>
									<!-- eslint-enable vue/no-v-html -->
									<div class="d-flex justify-content-between date-selection">
										<select v-model="selectedDay" class="form-select me-2">
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
											class="form-control ms-2 time-selection"
											:step="60 * 5"
											required
										/>
									</div>
								</div>
								<p class="mb-0">
									<span v-if="timeInThePast" class="text-danger">
										{{ $t("main.targetCharge.targetIsInThePast") }}
									</span>
									<span v-else-if="timeTooFarInTheFuture" class="text-secondary">
										{{ $t("main.targetCharge.targetIsTooFarInTheFuture") }}
									</span>
									&nbsp;
								</p>
								<TargetChargePlan
									v-if="targetChargePlanProps"
									v-bind="targetChargePlanProps"
								/>
							</div>
							<div class="modal-footer d-flex justify-content-between">
								<button
									type="button"
									class="btn btn-outline-secondary"
									data-bs-dismiss="modal"
									:disabled="!targetTime"
									@click="removeTargetTime"
								>
									{{ $t("main.targetCharge.remove") }}
								</button>
								<button
									type="submit"
									class="btn btn-primary"
									data-bs-dismiss="modal"
									:disabled="timeInThePast"
								>
									<span v-if="targetTime">
										{{ $t("main.targetCharge.update") }}
									</span>
									<span v-else>
										{{ $t("main.targetCharge.activate") }}
									</span>
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
import Modal from "bootstrap/js/dist/modal";
import "@h2d2/shopicons/es/filled/plus";
import "@h2d2/shopicons/es/filled/edit";
import LabelAndValue from "./LabelAndValue.vue";
import TargetChargePlan from "./TargetChargePlan.vue";
import api from "../api";

import formatter from "../mixins/formatter";

const DEFAULT_TARGET_TIME = "7:00";
const LAST_TARGET_TIME_KEY = "last_target_time";

export default {
	name: "TargetCharge",
	components: { LabelAndValue, TargetChargePlan },
	mixins: [formatter],
	props: {
		id: [String, Number],
		planActive: Boolean,
		targetTime: String,
		targetSoc: Number,
		targetEnergy: Number,
		socBasedCharging: Boolean,
		disabled: Boolean,
	},
	emits: ["target-time-updated", "target-time-removed"],
	data: function () {
		return {
			selectedDay: null,
			selectedTime: null,
			plan: {},
			tariff: {},
			modal: null,
			isModalVisible: false,
		};
	},
	computed: {
		targetChargeEnabled: function () {
			return this.targetTime;
		},
		timeInThePast: function () {
			const now = new Date();
			return now >= this.selectedTargetTime;
		},
		timeTooFarInTheFuture: function () {
			if (this.tariff?.rates) {
				const lastRate = this.tariff.rates[this.tariff.rates.length - 1];
				if (lastRate.end) {
					const end = new Date(lastRate.end);
					return this.selectedTargetTime >= end;
				}
			}
			return false;
		},
		selectedTargetTime: function () {
			return new Date(`${this.selectedDay}T${this.selectedTime || "00:00"}`);
		},
		modalId: function () {
			return `targetChargeModal_${this.id}`;
		},
		targetEnergyFormatted: function () {
			return this.fmtKWh(this.targetEnergy * 1e3, true, true, 1);
		},
		targetChargePlanProps: function () {
			const targetTime = this.selectedTargetTime;
			const { rates } = this.tariff;
			const { duration, unit, plan } = this.plan;
			return rates ? { duration, rates, plan, unit, targetTime } : null;
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
			this.updatePlan();
		},
		selectedTargetTime() {
			this.updatePlan();
		},
		targetSoc() {
			this.updatePlan();
		},
		targetEnergy() {
			this.updatePlan();
		},
		isModalVisible() {
			this.updatePlan();
		},
	},
	mounted() {
		this.modal = Modal.getOrCreateInstance(this.$refs.modal);
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hide.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs.modal.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.removeEventListener("hide.bs.modal", this.modalInvisible);
	},
	methods: {
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
		updatePlan: async function () {
			if (
				this.isModalVisible &&
				!this.timeInThePast &&
				(this.targetEnergy || this.targetSoc)
			) {
				try {
					const opts = {
						params: { targetTime: this.selectedTargetTime },
					};
					this.plan = (
						await api.get(`/loadpoints/${this.id}/target/plan`, opts)
					).data.result;
					this.tariff = (await api.get(`/tariff/planner`)).data.result;
				} catch (e) {
					console.error(e);
				}
			}
		},

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
		openModal() {
			this.modal.show();
			this.$nextTick(this.initInputFields);
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
@media (min-width: 992px) {
	.date-selection {
		width: 370px;
	}
}
.time-selection {
	flex-basis: 200px;
}
</style>
