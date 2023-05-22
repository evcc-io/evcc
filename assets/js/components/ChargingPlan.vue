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
						<div class="modal-body pt-2">
							<ul v-if="showTabs" class="nav nav-tabs">
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
										{{ $t("main.chargingPlan.arrivalTab") }} 🧪
									</a>
								</li>
							</ul>
							<div v-if="isModalVisible">
								<TargetCharge
									v-if="departureTabActive"
									v-bind="targetCharge"
									@target-time-updated="setTargetTime"
									@target-time-removed="removeTargetTime"
								/>
								<ChargingPlanArrival
									v-if="arrivalTabActive"
									v-bind="chargingPlanArrival"
									@minsoc-updated="setMinSoc"
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
import TargetCharge from "./TargetCharge.vue";
import ChargingPlanArrival from "./ChargingPlanArrival.vue";

import formatter from "../mixins/formatter";
import collector from "../mixins/collector";

export default {
	name: "ChargingPlan",
	components: { LabelAndValue, TargetCharge, ChargingPlanArrival },
	mixins: [formatter, collector],
	props: {
		id: [String, Number],
		planActive: Boolean,
		targetTime: String,
		targetSoc: Number,
		targetEnergy: Number,
		socBasedCharging: Boolean,
		disabled: Boolean,
		minSoc: Number,
		vehicleSoc: Number,
		vehicleName: String,
		smartCostLimit: Number,
		smartCostUnit: String,
	},
	emits: ["target-time-updated", "target-time-removed", "minsoc-updated"],
	data: function () {
		return {
			modal: null,
			isModalVisible: false,
			activeTab: "departure",
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
				return this.$t("main.chargingPlan.titleTargetCharge");
			}
			return this.$t("main.chargingPlan.title");
		},
		minSocEnabled: function () {
			return this.minSoc >= this.vehicleSoc && this.$hiddenFeatures();
		},
		departureTabActive: function () {
			return this.activeTab === "departure";
		},
		arrivalTabActive: function () {
			return this.activeTab === "arrival";
		},
		targetCharge: function () {
			return this.collectProps(TargetCharge);
		},
		chargingPlanArrival: function () {
			return this.collectProps(ChargingPlanArrival);
		},
		showTabs: function () {
			return this.$hiddenFeatures();
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
			if (this.minSocEnabled) {
				this.showArrivalTab();
			}
			if (this.targetChargeEnabled) {
				this.showDeatureTab();
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
		setTargetTime: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
			this.modal.hide();
		},
		removeTargetTime: function () {
			this.$emit("target-time-removed");
			this.modal.hide();
		},
		setMinSoc: function (minSoc) {
			this.$emit("minsoc-updated", minSoc);
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
