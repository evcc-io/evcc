<template>
	<div class="text-center">
		<LabelAndValue
			class="root flex-grow-1"
			:label="title"
			:class="disabled ? 'opacity-0' : 'opacity-1'"
		>
			<h3 class="value m-0 d-block d-sm-flex align-items-baseline justify-content-center">
				<button
					class="value-button p-0"
					:class="enabled ? 'evcc-default-text' : 'text-gray'"
					:disabled="disabled"
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
								{{ $t("main.chargingPlan.modalTitle") }}
							</h5>
							<button
								type="button"
								class="btn-close"
								data-bs-dismiss="modal"
								aria-label="Close"
							></button>
						</div>
						<form @submit.prevent="setTargetTime">
							<div class="modal-body pt-2">
								<ul class="nav nav-tabs">
									<li class="nav-item">
										<a
											class="nav-link"
											:class="{ active: timeTabActive }"
											href="#"
											@click.prevent="showTimeTab"
										>
											Depature
										</a>
									</li>
									<li v-if="smartCostTabAvailable && false" class="nav-item">
										<a
											class="nav-link"
											:class="{ active: smartCostTabActive }"
											href="#"
											@click.prevent="showSmartCostTab"
										>
											<div v-if="co2Available">
												Green energy
												<span class="badge bg-secondary">&leq; 750g</span>
											</div>
											<div v-else>
												Cheap
												<span
													class="badge bg-secondary d-none d-sm-inline-block"
													>&leq; 0,23ct</span
												>
											</div>
										</a>
									</li>
									<li class="nav-item">
										<a
											class="nav-link"
											:class="{ active: minSocTabActive }"
											href="#"
											@click.prevent="showMinSocTab"
										>
											Arrival
										</a>
									</li>
								</ul>
								<TargetCharge v-if="timeTabActive" v-bind="targetCharge" />
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
import LabelAndValue from "./LabelAndValue.vue";
import TargetCharge from "./TargetCharge.vue";

import formatter from "../mixins/formatter";
import collector from "../mixins/collector";

export default {
	name: "ChargingPlan",
	components: { LabelAndValue, TargetCharge },
	mixins: [formatter, collector],
	props: {
		id: [String, Number],
		planActive: Boolean,
		targetTime: String,
		targetSoc: Number,
		targetEnergy: Number,
		socBasedCharging: Boolean,
		disabled: Boolean,
		smartCostLimit: Number,
		tariffPlannerUnit: String,
		tariffGrid: Number,
		tariffCo2: Number,
		minSoc: Number,
		vehicleSoc: Number,
	},
	emits: ["target-time-updated", "target-time-removed"],
	data: function () {
		return {
			modal: null,
			isModalVisible: false,
			activeTab: "time",
		};
	},
	computed: {
		targetChargeEnabled: function () {
			return this.targetTime;
		},
		enabled: function () {
			return this.targetChargeEnabled || this.minSocEnabled;
		},
		minSocLabel: function () {
			return `${Math.round(this.minSoc)} %`;
		},
		modalId: function () {
			return `chargingPlanModal_${this.id}`;
		},
		title: function () {
			if (this.minSocEnabled) {
				return this.$t("main.chargingPlan.titleMinSoc");
			}
			if (this.targetChargeEnabled) {
				return this.$t("main.chargingPlan.titleTargetCharge");
			}
			return this.$t("main.chargingPlan.title");
		},
		smartCostLabelNow: function () {
			if (this.co2Available && this.tariffCo2) {
				return `now ${this.fmtCo2Short(this.tariffCo2)}`;
			} else if (this.tariffGrid) {
				return `now ${this.fmtPricePerKWh(this.tariffGrid, this.tariffPlannerUnit, true)}`;
			}
			return "";
		},
		minSocEnabled: function () {
			return this.minSoc >= this.vehicleSoc;
		},
		co2Available: function () {
			return this.tariffPlannerUnit === "gCO2eq";
		},
		timeTabActive: function () {
			return this.activeTab === "time";
		},
		smartCostTabActive: function () {
			return this.activeTab === "smartcost";
		},
		minSocTabActive: function () {
			return this.activeTab === "minsoc";
		},
		targetCharge: function () {
			return this.collectProps(TargetCharge);
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
		openModal() {
			this.modal.show();
			this.$nextTick(this.initInputFields);
		},
		// not computed because it needs to update over time
		targetTimeLabel: function () {
			const targetDate = new Date(this.targetTime);
			return this.$t("main.chargingPlan.activeLabel", {
				time: this.fmtAbsoluteDate(targetDate),
			});
		},
		showTimeTab: function () {
			this.activeTab = "time";
		},
		showSmartCostTab: function () {
			this.activeTab = "smartcost";
		},
		showMinSocTab: function () {
			this.activeTab = "minsoc";
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
.extraValue {
	color: var(--evcc-gray);
	font-size: 14px;
}
</style>
