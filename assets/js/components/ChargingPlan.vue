<template>
	<div class="text-center">
		<LabelAndValue
			class="root flex-grow-1"
			:label="title"
			:class="disabled ? 'opacity-25' : 'opacity-100'"
			data-testid="charging-plan"
		>
			<h3 class="value m-0 d-block align-items-baseline justify-content-center">
				<button
					class="value-button p-0"
					:class="enabled ? 'evcc-default-text' : 'text-gray'"
					@click="openModal"
				>
					<strong v-if="minSocEnabled">{{ minSocLabel }}</strong>
					<strong v-else-if="targetChargeEnabled">{{ targetTimeLabel() }}</strong>
					<span v-else>{{ $t("main.chargingPlan.none") }}</span>
				</button>
			</h3>
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
								}}<span v-if="vehicle">: {{ vehicle.title }}</span>
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
									@plan-added="addPlan"
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

export default {
	name: "ChargingPlan",
	components: { LabelAndValue, ChargingPlanSettings, ChargingPlanArrival },
	mixins: [formatter, collector],
	props: {
		id: [String, Number],
		planActive: Boolean,
		targetTime: String,
		effectiveLimitSoc: Number,
		limitEnergy: Number,
		socBasedCharging: Boolean,
		disabled: Boolean,
		vehicle: Object,
		vehicleSoc: Number,
		vehicleName: String,
		smartCostLimit: Number,
		smartCostType: String,
		currency: String,
		mode: String,
		vehicleCapacity: Number,
		rangePerSoc: Number,
	},
	data: function () {
		return {
			modal: null,
			isModalVisible: false,
			activeTab: "departure",
		};
	},
	computed: {
		minSoc: function () {
			return this.vehicle?.minSoc;
		},
		limitSoc: function () {
			return this.vehicle?.limitSoc;
		},
		plans: function () {
			return this.vehicle?.plans;
		},
		targetChargeEnabled: function () {
			return this.vehicle?.plan?.length > 0;
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
			if (this.targetChargeEnabled) {
				return this.$t("main.chargingPlan.titleChargingPlanSettings");
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
		// not computed because it needs to update over time
		targetTimeLabel: function () {
			const targetDate = new Date(this.targetTime);
			return this.$t("main.chargingPlan.activeLabel", {
				time: this.fmtAbsoluteDate(targetDate),
			});
		},
		showDeatureTab: function () {
			this.activeTab = "departure";
		},
		showArrivalTab: function () {
			this.activeTab = "arrival";
		},
		apiPath: function (func) {
			return "vehicles/" + this.vehicle.name + "/" + func;
		},
		addPlan: function (plan) {
			const soc = plan.soc;
			const time = plan.time.toISOString();
			api.post(this.apiPath("plan/soc/") + `${soc}/${time}`);
		},
		setMinSoc: function (soc) {
			api.post(this.apiPath("minsoc") + `/${soc}`);
		},
		setLimitSoc: function (soc) {
			api.post(this.apiPath("limitsoc") + `/${soc}`);
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
	text-decoration: underline;
}
.root {
	transition: opacity var(--evcc-transition-medium) linear;
}
.value:hover {
	color: var(--bs-color-white);
}
</style>
