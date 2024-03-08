<template>
	<div class="text-center">
		<LabelAndValue
			class="root flex-grow-1"
			:label="title"
			:class="disabled ? 'opacity-25' : 'opacity-100'"
			data-testid="charging-plan"
		>
			<div class="value m-0 d-block align-items-baseline justify-content-center">
				<button class="value-button p-0" :class="buttonColor" @click="openModal">
					<strong v-if="minSocEnabled" class="text-decoration-underline">
						{{ minSocLabel }}
					</strong>
					<strong v-else-if="targetChargeEnabled">
						<span class="targetTimeLabel"> {{ targetTimeLabel() }}</span>
						<div
							class="extraValue text-nowrap"
							:class="{ 'text-warning': planOverrun }"
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
								<ChargingPlanSettings
									v-if="departureTabActive"
									v-bind="chargingPlanSettingsProps"
									@plan-updated="updatePlan"
									@plan-removed="removePlan"
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
import ChargingPlanSettings from "./ChargingPlanSettings.vue";
import ChargingPlanArrival from "./ChargingPlanArrival.vue";

import formatter from "../mixins/formatter";
import collector from "../mixins/collector";
import api from "../api";
import { optionStep, fmtEnergy } from "../utils/energyOptions";

export default {
	name: "ChargingPlan",
	components: { LabelAndValue, ChargingPlanSettings, ChargingPlanArrival },
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
		planOverrun: Boolean,
		rangePerSoc: Number,
		smartCostLimit: Number,
		smartCostType: String,
		socBasedPlanning: Boolean,
		socBasedCharging: Boolean,
		socPerKwh: Number,
		vehicle: Object,
		capacity: Number,
		vehicleSoc: Number,
		vehicleTargetSoc: Number,
	},
	data: function () {
		return {
			modal: null,
			isModalVisible: false,
			activeTab: "departure",
		};
	},
	computed: {
		buttonColor: function () {
			if (this.planOverrun) {
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
		plans: function () {
			if (this.socBasedPlanning) {
				return this.vehicle?.plans || [];
			}
			if (this.planEnergy && this.planTime) {
				return [{ energy: this.planEnergy, time: this.planTime }];
			}
			return [];
		},
		targetChargeEnabled: function () {
			return this.effectivePlanTime;
		},
		enabled: function () {
			return this.targetChargeEnabled || this.minSocEnabled;
		},
		minSocLabel: function () {
			return `${Math.round(this.minSoc)}%`;
		},
		modalId: function () {
			return `chargingPlanModal_${this.id}`;
		},
		title: function () {
			if (this.minSocEnabled) {
				return this.$t("main.chargingPlan.titleMinSoc");
			}
			return this.$t("main.chargingPlan.title");
		},
		minSocEnabled: function () {
			return this.minSoc > this.vehicleSoc;
		},
		departureTabActive: function () {
			return this.activeTab === "departure";
		},
		arrivalTabActive: function () {
			return this.activeTab === "arrival";
		},
		chargingPlanSettingsProps: function () {
			return this.collectProps(ChargingPlanSettings);
		},
		chargingPlanArrival: function () {
			return this.collectProps(ChargingPlanArrival);
		},
		targetSocLabel: function () {
			if (this.socBasedPlanning) {
				return `${Math.round(this.effectivePlanSoc)}%`;
			}
			return fmtEnergy(
				this.planEnergy,
				optionStep(this.capacity || 100),
				this.fmtKWh,
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
	mounted() {
		this.modal = Modal.getOrCreateInstance(this.$refs.modal);
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hidden.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal?.removeEventListener("hidden.bs.modal", this.modalInvisible);
	},
	methods: {
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
		openModal() {
			this.showDeatureTab();
			if (this.minSocEnabled) {
				this.showArrivalTab();
			}
			this.modal.show();
		},
		openPlanModal() {
			this.showDeatureTab();
			this.modal.show();
		},
		// not computed because it needs to update over time
		targetTimeLabel: function () {
			const targetDate = new Date(this.effectivePlanTime);
			return this.fmtAbsoluteDate(targetDate);
		},
		showDeatureTab: function () {
			this.activeTab = "departure";
		},
		showArrivalTab: function () {
			this.activeTab = "arrival";
		},
		updatePlan: function ({ soc, time, energy }) {
			const timeISO = time.toISOString();
			if (this.socBasedPlanning) {
				api.post(`${this.apiVehicle}plan/soc/${soc}/${timeISO}`);
			} else {
				api.post(`${this.apiLoadpoint}plan/energy/${energy}/${timeISO}`);
			}
		},
		removePlan: function () {
			if (this.socBasedPlanning) {
				api.delete(`${this.apiVehicle}plan/soc`);
			} else {
				api.delete(`${this.apiLoadpoint}plan/energy`);
			}
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
