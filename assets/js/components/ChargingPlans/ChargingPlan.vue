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

<script lang="ts">
import Modal from "bootstrap/js/dist/modal";
import LabelAndValue from "../Helper/LabelAndValue.vue";
import PlansSettings from "./PlansSettings.vue";
import Arrival from "./Arrival.vue";

import formatter from "@/mixins/formatter";
import collector from "@/mixins/collector";
import api from "@/api";
import { optionStep, fmtEnergy } from "@/utils/energyOptions";
import { defineComponent, type PropType } from "vue";
import type { CURRENCY, Timeout, Vehicle } from "@/types/evcc";
import type { StaticPlan, StaticSocPlan, StaticEnergyPlan, RepeatingPlan } from "./types";
import type { Forecast } from "@/types/evcc.ts";
const ONE_MINUTE = 60 * 1000;

export default defineComponent({
	name: "ChargingPlan",
	components: {
		LabelAndValue,
		ChargingPlansSettings: PlansSettings,
		ChargingPlanArrival: Arrival,
	},
	mixins: [formatter, collector],
	props: {
		currency: String as PropType<CURRENCY>,
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
		planPrecondition: { type: Number, default: 0 },
		planOverrun: Number,
		rangePerSoc: Number,
		smartCostType: String,
		socBasedPlanning: Boolean,
		socBasedCharging: Boolean,
		socPerKwh: Number,
		vehicle: Object as PropType<Vehicle>,
		capacity: Number,
		vehicleSoc: Number,
		vehicleLimitSoc: Number,
		forecast: Object as PropType<Forecast>,
	},
	data() {
		return {
			modal: null as Modal | null,
			isModalVisible: false,
			activeTab: "departure",
			targetTimeLabel: "",
			interval: null as Timeout,
		};
	},
	computed: {
		buttonColor(): string {
			if (this.planTimeUnreachable) {
				return "text-warning";
			}
			if (!this.enabled) {
				return "text-gray";
			}
			return "evcc-default-text";
		},
		minSoc(): number | undefined {
			return this.vehicle?.minSoc;
		},
		limitSoc(): number | undefined {
			return this.vehicle?.limitSoc;
		},
		staticPlan(): StaticPlan | null {
			if (this.socBasedPlanning) {
				const plan = this.vehicle?.plan as StaticSocPlan;
				if (plan) {
					return {
						soc: plan.soc,
						time: new Date(plan.time),
						precondition: plan.precondition,
					};
				}
				return null;
			}
			if (this.planEnergy && this.planTime) {
				return {
					energy: this.planEnergy,
					time: new Date(this.planTime),
					precondition: this.planPrecondition,
				};
			}
			return null;
		},
		repeatingPlans(): RepeatingPlan[] {
			if (this.vehicle && this.vehicle.repeatingPlans.length > 0) {
				return [...this.vehicle.repeatingPlans];
			}
			return [];
		},
		enabled(): boolean {
			return !!this.effectivePlanTime;
		},
		modalId(): string {
			return `chargingPlanModal_${this.id}`;
		},
		departureTabActive(): boolean {
			return this.activeTab === "departure";
		},
		arrivalTabActive(): boolean {
			return this.activeTab === "arrival";
		},
		chargingPlansSettingsProps(): any {
			return this.collectProps(PlansSettings);
		},
		chargingPlanArrival(): any {
			return this.collectProps(Arrival);
		},
		targetSocLabel(): string {
			if (this.socBasedPlanning && this.effectivePlanSoc) {
				return this.fmtPercentage(this.effectivePlanSoc);
			}
			return fmtEnergy(
				this.planEnergy,
				optionStep(this.capacity || 100),
				this.fmtWh,
				this.$t("main.targetEnergy.noLimit")
			);
		},
		apiVehicle(): string {
			return `vehicles/${this.vehicle?.name}/`;
		},
		apiLoadpoint(): string {
			return `loadpoints/${this.id}/`;
		},
	},
	watch: {
		effectivePlanTime(): void {
			this.updateTargetTimeLabel();
		},
		"$i18n.locale": {
			handler(): void {
				this.updateTargetTimeLabel();
			},
		},
	},
	mounted(): void {
		const ref = this.$refs["modal"];
		if (ref) {
			this.modal = Modal.getOrCreateInstance(ref);
			ref.addEventListener("show.bs.modal", this.modalVisible);
			ref.addEventListener("hidden.bs.modal", this.modalInvisible);
			ref.addEventListener("hide.bs.modal", this.checkUnsavedOnClose);
		}
		this.interval = setInterval(this.updateTargetTimeLabel, ONE_MINUTE);
		this.updateTargetTimeLabel();
	},
	unmounted(): void {
		const ref = this.$refs["modal"];
		if (ref) {
			ref.removeEventListener("show.bs.modal", this.modalVisible);
			ref.removeEventListener("hidden.bs.modal", this.modalInvisible);
			ref.removeEventListener("hide.bs.modal", this.checkUnsavedOnClose);
		}
		if (this.interval) {
			clearInterval(this.interval);
		}
	},
	methods: {
		checkUnsavedOnClose(): void {
			const applyButton = this.$refs["modal"]?.querySelector<HTMLElement>(
				"[data-testid=plan-apply]"
			);
			if (applyButton) {
				if (confirm(this.$t("main.chargingPlan.unsavedChanges"))) {
					applyButton.click();
				}
			}
		},
		modalVisible(): void {
			this.isModalVisible = true;
		},
		modalInvisible(): void {
			this.isModalVisible = false;
		},
		openModal(): void {
			this.showDeatureTab();
			this.modal?.show();
		},
		openPlanModal(arrivalTab = false) {
			if (arrivalTab) {
				this.showArrivalTab();
			} else {
				this.showDeatureTab();
			}
			this.modal?.show();
		},
		updateTargetTimeLabel(): void {
			if (!this.effectivePlanTime) return;
			const targetDate = new Date(this.effectivePlanTime);
			this.targetTimeLabel = this.fmtAbsoluteDate(targetDate);
		},
		showDeatureTab(): void {
			this.activeTab = "departure";
		},
		showArrivalTab(): void {
			this.activeTab = "arrival";
		},
		updateStaticPlan(plan: StaticPlan): void {
			const timeISO = plan.time.toISOString();
			const params = plan.precondition ? { precondition: plan.precondition } : undefined;
			if (this.socBasedPlanning) {
				const p = plan as StaticSocPlan;
				api.post(`${this.apiVehicle}plan/soc/${p.soc}/${timeISO}`, null, { params });
			} else {
				const p = plan as StaticEnergyPlan;
				api.post(`${this.apiLoadpoint}plan/energy/${p.energy}/${timeISO}`, null, {
					params,
				});
			}
		},
		removeStaticPlan(): void {
			if (this.socBasedPlanning) {
				api.delete(`${this.apiVehicle}plan/soc`);
			} else {
				api.delete(`${this.apiLoadpoint}plan/energy`);
			}
		},
		updateRepeatingPlans(plans: RepeatingPlan[]): void {
			api.post(`${this.apiVehicle}plan/repeating`, { plans });
		},
		setMinSoc(soc: number): void {
			api.post(`${this.apiVehicle}minsoc/${soc}`);
		},
		setLimitSoc(soc: number): void {
			api.post(`${this.apiVehicle}limitsoc/${soc}`);
		},
	},
});
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
