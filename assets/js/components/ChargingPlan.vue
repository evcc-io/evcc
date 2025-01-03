<template>
	<div class="text-center">
		<LabelAndValue
			class="root flex-grow-1"
			:label="$t('main.chargingPlan.title')"
			:class="disabled ? 'opacity-25' : 'opacity-100'"
			data-testid="charging-plan"
		>
			<div class="value m-0 d-block align-items-baseline justify-content-center">
				<button
					class="value-button p-0"
					:class="buttonColor"
					data-testid="charging-plan-button"
					@click="openModal"
				>
					<strong v-if="enabled">
						<span class="targetTimeLabel"> {{ targetTimeLabel }}</span>
						<div
							class="extraValue text-nowrap"
							:class="{ 'text-warning': planTimeUnreachable }"
						>
							{{ targetSocLabel }}
						</div>
					</strong>
					<span v-else class="text-decoration-underline">
						{{ $t("main.chargingPlan.none") }}
					</span>
				</button>
			</div>
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
				data-testid="charging-plan-modal"
			>
				<div class="modal-dialog modal-dialog-centered" role="document">
					<div class="modal-content">
						<div class="modal-header">
							<h5 class="modal-title">
								{{ $t("main.chargingPlan.modalTitle")
								}}<span v-if="socBasedPlanning && vehicle"
									>: {{ vehicle.title }}</span
								>
							</h5>
							<button
								type="button"
								class="btn-close"
								data-bs-dismiss="modal"
								aria-label="Close"
							></button>
						</div>
						<div class="modal-body pt-2">
							<ul class="nav nav-tabs">
								<li class="nav-item">
									<a
										class="nav-link"
										:class="{ active: departureTabActive }"
										href="#"
										@click.prevent="showDeatureTab"
									>
										{{ $t("main.chargingPlan.departureTab") }}
									</a>
								</li>
								<li class="nav-item">
									<a
										class="nav-link"
										:class="{ active: arrivalTabActive }"
										href="#"
										@click.prevent="showArrivalTab"
									>
										{{ $t("main.chargingPlan.arrivalTab") }}
									</a>
								</li>
							</ul>
							<div v-if="isModalVisible">
								<ChargingPlansSettings
									v-if="departureTabActive"
									v-bind="chargingPlansSettingsProps"
									@static-plan-updated="updateStaticPlan"
									@static-plan-removed="removeStaticPlan"
									@repeating-plans-updated="updateRepeatingPlans"
								/>
								<ChargingPlanArrival
									v-if="arrivalTabActive"
									v-bind="chargingPlanArrival"
									@minsoc-updated="setMinSoc"
									@limitsoc-updated="setLimitSoc"
								/>
							</div>
						</div>
					</div>
				</div>
			</div>
		</Teleport>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import LabelAndValue from "./LabelAndValue.vue";
import ChargingPlansSettings from "./ChargingPlansSettings.vue";
import ChargingPlanArrival from "./ChargingPlanArrival.vue";

import formatter from "../mixins/formatter";
import collector from "../mixins/collector";
import api from "../api";
import { optionStep, fmtEnergy } from "../utils/energyOptions";

const ONE_MINUTE = 60 * 1000;

export default {
	name: "ChargingPlan",
	components: { LabelAndValue, ChargingPlansSettings, ChargingPlanArrival },
	mixins: [formatter, collector],
	props: {
		currency: String,
		disabled: Boolean,
		effectiveLimitSoc: Number,
		effectivePlanSoc: Number,
		effectivePlanTime: String,
		id: [String, Number],
		limitEnergy: Number,
		mode: String,
		planActive: Boolean,
		planEnergy: Number,
		planTime: String,
		planTimeUnreachable: Boolean,
		planOverrun: Number,
		rangePerSoc: Number,
		smartCostType: String,
		socBasedPlanning: Boolean,
		socBasedCharging: Boolean,
		socPerKwh: Number,
		vehicle: Object,
		capacity: Number,
		vehicleSoc: Number,
		vehicleLimitSoc: Number,
	},
	data: function () {
		return {
			modal: null,
			isModalVisible: false,
			activeTab: "departure",
			targetTimeLabel: "",
			interval: null,
		};
	},
	computed: {
		buttonColor: function () {
			if (this.planTimeUnreachable) {
				return "text-warning";
			}
			if (!this.enabled) {
				return "text-gray";
			}
			return "evcc-default-text";
		},
		minSoc: function () {
			return this.vehicle?.minSoc;
		},
		limitSoc: function () {
			return this.vehicle?.limitSoc;
		},
		staticPlan: function () {
			if (this.socBasedPlanning) {
				return this.vehicle?.plan;
			}
			if (this.planEnergy && this.planTime) {
				return { energy: this.planEnergy, time: this.planTime };
			}
			return null;
		},
		repeatingPlans: function () {
			if (this.vehicle?.repeatingPlans.length > 0) {
				return [...this.vehicle.repeatingPlans];
			}
			return [];
		},
		enabled: function () {
			return this.effectivePlanTime;
		},
		modalId: function () {
			return `chargingPlanModal_${this.id}`;
		},
		departureTabActive: function () {
			return this.activeTab === "departure";
		},
		arrivalTabActive: function () {
			return this.activeTab === "arrival";
		},
		chargingPlansSettingsProps: function () {
			return this.collectProps(ChargingPlansSettings);
		},
		chargingPlanArrival: function () {
			return this.collectProps(ChargingPlanArrival);
		},
		targetSocLabel: function () {
			if (this.socBasedPlanning) {
				return this.fmtPercentage(this.effectivePlanSoc);
			}
			return fmtEnergy(
				this.planEnergy,
				optionStep(this.capacity || 100),
				this.fmtWh,
				this.$t("main.targetEnergy.noLimit")
			);
		},
		apiVehicle: function () {
			return `vehicles/${this.vehicle?.name}/`;
		},
		apiLoadpoint: function () {
			return `loadpoints/${this.id}/`;
		},
	},
	watch: {
		effectivePlanTime() {
			this.updateTargetTimeLabel();
		},
		"$i18n.locale": {
			handler() {
				this.updateTargetTimeLabel();
			},
		},
	},
	mounted() {
		this.modal = Modal.getOrCreateInstance(this.$refs.modal);
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hidden.bs.modal", this.modalInvisible);
		this.$refs.modal.addEventListener("hide.bs.modal", this.checkUnsavedOnClose);
		this.interval = setInterval(this.updateTargetTimeLabel, ONE_MINUTE);
		this.updateTargetTimeLabel();
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal?.removeEventListener("hidden.bs.modal", this.modalInvisible);
		this.$refs.modal?.removeEventListener("hide.bs.modal", this.checkUnsavedOnClose);
		clearInterval(this.interval);
	},
	methods: {
		checkUnsavedOnClose: function () {
			const $applyButton = this.$refs.modal.querySelector("[data-testid=plan-apply]");
			if ($applyButton) {
				if (confirm(this.$t("main.chargingPlan.unsavedChanges"))) {
					$applyButton.click();
				}
			}
		},
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
		openModal() {
			this.showDeatureTab();
			this.modal.show();
		},
		openPlanModal(arrivalTab = false) {
			if (arrivalTab) {
				this.showArrivalTab();
			} else {
				this.showDeatureTab();
			}
			this.modal.show();
		},
		updateTargetTimeLabel: function () {
			if (!this.effectivePlanTime) return "";
			const targetDate = new Date(this.effectivePlanTime);
			this.targetTimeLabel = this.fmtAbsoluteDate(targetDate);
		},
		showDeatureTab: function () {
			this.activeTab = "departure";
		},
		showArrivalTab: function () {
			this.activeTab = "arrival";
		},
		updateStaticPlan: function ({ soc, time, energy }) {
			const timeISO = time.toISOString();
			if (this.socBasedPlanning) {
				api.post(`${this.apiVehicle}plan/soc/${soc}/${timeISO}`);
			} else {
				api.post(`${this.apiLoadpoint}plan/energy/${energy}/${timeISO}`);
			}
		},
		removeStaticPlan: function () {
			if (this.socBasedPlanning) {
				api.delete(`${this.apiVehicle}plan/soc`);
			} else {
				api.delete(`${this.apiLoadpoint}plan/energy`);
			}
		},
		updateRepeatingPlans: function (plans) {
			api.post(`${this.apiVehicle}plan/repeating`, { plans });
		},
		setMinSoc: function (soc) {
			api.post(`${this.apiVehicle}minsoc/${soc}`);
		},
		setLimitSoc: function (soc) {
			api.post(`${this.apiVehicle}limitsoc/${soc}`);
		},
	},
};
</script>

<style scoped>
.value {
	line-height: 1.2;
	border: none;
}
.value-button {
	font-size: 18px;
	border: none;
	background: none;
}
.root {
	transition: opacity var(--evcc-transition-medium) linear;
}
.value:hover {
	color: var(--bs-color-white);
}
.extraValue {
	color: var(--evcc-gray);
	font-size: 14px;
	text-decoration: none;
}
.targetTimeLabel {
	text-decoration: underline;
}
</style>
