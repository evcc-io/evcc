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
					<strong v-else-if="smartCostEnabled">{{ smartCostLabel }}</strong>
					<span v-else>{{ $t("main.smartCharging.none") }}</span>
				</button>
				<div v-if="smartCostEnabled" class="extraValue ms-0 ms-sm-1 text-nowrap">
					{{ smartCostLabelNow }}
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
								{{ $t("main.smartCharging.modalTitle") }}
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
								<!--
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
									<li v-if="smartCostTabAvailable" class="nav-item">
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
											Min range
										</a>
									</li>
								</ul>
								<TargetCharge v-if="timeTabActive" v-bind="targetCharge" />
								-->
								<div class="accordion accordion-flush" id="accordionFlushExample">
									<div class="accordion-item">
										<h2 class="accordion-header" id="flush-headingOne">
											<button
												class="accordion-button"
												type="button"
												data-bs-toggle="collapse"
												data-bs-target="#flush-collapseOne"
												aria-expanded="true"
												aria-controls="flush-collapseOne"
											>
												<div
													class="d-flex justify-content-between flex-grow-1 me-3"
												>
													Depature time
													<span class="badge rounded-pill bg-success"
														>Fr. 7:30</span
													>
												</div>
											</button>
										</h2>
										<div
											id="flush-collapseOne"
											class="accordion-collapse collaps show pb-3"
											aria-labelledby="flush-headingOne"
											data-bs-parent="#accordionFlushExample"
										>
											<TargetCharge
												v-if="timeTabActive"
												v-bind="targetCharge"
											/>
										</div>
									</div>
									<div class="accordion-item">
										<h2 class="accordion-header" id="flush-headingTwo">
											<button
												class="accordion-button collapsed"
												type="button"
												data-bs-toggle="collapse"
												data-bs-target="#flush-collapseTwo"
												aria-expanded="false"
												aria-controls="flush-collapseTwo"
											>
												<div
													class="d-flex justify-content-between flex-grow-1 me-3"
												>
													Green energy
													<span class="badge bg-secondary"
														>&leq; 750g</span
													>
												</div>
											</button>
										</h2>
										<div
											id="flush-collapseTwo"
											class="accordion-collapse collapse"
											aria-labelledby="flush-headingTwo"
											data-bs-parent="#accordionFlushExample"
										>
											<div class="accordion-body">
												Placeholder content for this accordion, which is
												intended to demonstrate the
												<code>.accordion-flush</code> class. This is the
												second item's accordion body. Let's imagine this
												being filled with some actual content.
											</div>
										</div>
									</div>
									<div class="accordion-item">
										<h2 class="accordion-header" id="flush-headingThree">
											<button
												class="accordion-button collapsed"
												type="button"
												data-bs-toggle="collapse"
												data-bs-target="#flush-collapseThree"
												aria-expanded="false"
												aria-controls="flush-collapseThree"
											>
												Minimum range
											</button>
										</h2>
										<div
											id="flush-collapseThree"
											class="accordion-collapse collapse"
											aria-labelledby="flush-headingThree"
											data-bs-parent="#accordionFlushExample"
										>
											<div class="accordion-body">
												Placeholder content for this accordion, which is
												intended to demonstrate the
												<code>.accordion-flush</code> class. This is the
												third item's accordion body. Nothing more exciting
												happening here in terms of content, but just filling
												up the space to make it look, at least at first
												glance, a bit more representative of how this would
												look in a real-world application.
											</div>
										</div>
									</div>
								</div>
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
	name: "SmartCharging",
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
		smartCostUnit: String,
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
		smartCostEnabled: function () {
			return this.smartCostLimit && this.smartCostLimit != 0;
		},
		enabled: function () {
			return this.targetChargeEnabled || this.smartCostEnabled || this.minSocEnabled;
		},
		smartCostLabel: function () {
			const price = this.co2Available
				? this.fmtCo2Short(this.smartCostLimit)
				: this.fmtPricePerKWh(this.smartCostLimit, this.smartCostUnit, true);
			return `< ${price}`;
		},
		minSocLabel: function () {
			return `${Math.round(this.minSoc)} %`;
		},
		modalId: function () {
			return `smartChargingModal_${this.id}`;
		},
		title: function () {
			if (this.minSocEnabled) {
				return this.$t("main.smartCharging.titleMinSoc");
			}
			if (this.targetChargeEnabled) {
				return this.$t("main.smartCharging.titleTargetCharge");
			}
			if (this.smartCostEnabled) {
				if (this.co2Available) {
					return this.$t("main.smartCharging.titleCo2");
				} else {
					return this.$t("main.smartCharging.titlePrice");
				}
			}
			return this.$t("main.smartCharging.title");
		},
		smartCostLabelNow: function () {
			if (this.co2Available && this.tariffCo2) {
				return `now ${this.fmtCo2Short(this.tariffCo2)}`;
			} else if (this.tariffGrid) {
				return `now ${this.fmtPricePerKWh(this.tariffGrid, this.smartCostUnit, true)}`;
			}
			return "";
		},
		smartCostTabAvailable: function () {
			return this.dynamicPricesAvailabe || this.co2Available;
		},
		dynamicPricesAvailabe: function () {
			// TODO: determin if dynamic prices exist
			return true;
		},
		minSocEnabled: function () {
			return this.minSoc >= this.vehicleSoc;
		},
		co2Available: function () {
			return this.smartCostUnit === "gCO2eq";
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
			return this.$t("main.smartCharging.activeLabel", {
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
.modal-content {
	padding: 1.25rem 0 !important;
}
.modal-header {
	padding: 0 1rem 1rem;
}
.accordion-collapse {
	padding: 0 1.25rem;
}
</style>
