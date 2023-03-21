<template>
	<div>
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
					<strong v-if="targetChargeEnabled">{{ targetTimeLabel() }}</strong>
					<strong v-else-if="smartCostEnabled">{{ smartCostLabel }}</strong>
					<span v-else>{{ $t("main.smartCharging.none") }}</span>
				</button>
				<div v-if="smartCostEnabled" class="extraValue ms-0 ms-sm-1 text-nowrap">
					now 23 ct
				</div>
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
								<ul class="nav nav-tabs">
									<li class="nav-item">
										<a
											class="nav-link"
											:class="{ active: timeTabActive }"
											href="#"
											@click.prevent="showTimeTab"
										>
											Depature
											<span
												class="badge rounded-pill bg-success d-none d-sm-inline-block"
												>Fr. 7:30</span
											>
										</a>
									</li>
									<li v-if="priceTabAvailable" class="nav-item">
										<a
											class="nav-link"
											:class="{ active: priceTabActive }"
											href="#"
											@click.prevent="showPriceTab"
										>
											Cheap
											<span
												class="badge bg-secondary d-none d-sm-inline-block"
												>&leq; 0,23ct</span
											>
										</a>
									</li>
									<li v-if="co2TabAvailable" class="nav-item">
										<a
											class="nav-link"
											:class="{ active: co2TabActive }"
											href="#"
											@click.prevent="showCo2Tab"
										>
											Green energy
											<span class="badge bg-secondary">&leq; 750g</span>
										</a>
									</li>
									<li class="nav-item">
										<a
											class="nav-link"
											:class="{ active: minSocTabActive }"
											href="#"
											@click.prevent="showMinSocTab"
										>
											Min range
										</a>
									</li>
								</ul>
								<TargetCharge v-if="timeTabActive" />
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

export default {
	name: "SmartCharging",
	components: { LabelAndValue, TargetCharge },
	mixins: [formatter],
	props: {
		id: [String, Number],
		planActive: Boolean,
		targetTime: String,
		targetSoc: Number,
		targetEnergy: Number,
		socBasedCharging: Boolean,
		disabled: Boolean,
		smartCostLimit: Number,
		smartCostUnit: String,
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
		smartCostEnabled: function () {
			return this.smartCostLimit && this.smartCostLimit != 0;
		},
		enabled: function () {
			return this.targetChargeEnabled || this.smartCostEnabled;
		},
		smartCostLabel: function () {
			const price = this.fmtPricePerKWh(this.smartCostLimit, "EUR", true);
			return `< ${price}`;
		},
		modalId: function () {
			return `smartChargingModal_${this.id}`;
		},
		title: function () {
			if (this.targetChargeEnabled) {
				return this.$t("main.smartCharging.titleTargetCharge");
			}
			if (this.smartCostEnabled) {
				return this.$t("main.smartCharging.titleSmartCost");
			}
			return this.$t("main.smartCharging.title");
		},
		priceTabAvailable: function () {
			return !this.co2TabAvailable;
		},
		co2TabAvailable: function () {
			return this.smartCostUnit === "gCO2eq";
		},
		timeTabActive: function () {
			return this.activeTab === "time";
		},
		co2TabActive: function () {
			return this.activeTab === "co2";
		},
		priceTabActive: function () {
			return this.activeTab === "price";
		},
		minSocTabActive: function () {
			return this.activeTab === "minsoc";
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
			return this.$t("main.smartCharging.activeLabel", {
				time: this.fmtAbsoluteDate(targetDate),
			});
		},
		showTimeTab: function () {
			this.activeTab = "time";
		},
		showPriceTab: function () {
			this.activeTab = "price";
		},
		showCo2Tab: function () {
			this.activeTab = "co2";
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
