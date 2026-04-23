<template>
	<Teleport to="body">
		<GenericModal
			id="chargingPlanModal"
			ref="modal"
			:title="modalTitle"
			size="xl"
			data-testid="charging-plan-modal"
			@open="modalVisible"
			@closed="modalInvisible"
		>
			<div class="pt-2">
				<ul class="nav nav-tabs">
					<li class="nav-item">
						<a
							class="nav-link"
							:class="{ active: departureTabActive }"
							href="#"
							@click.prevent="showDepartureTab"
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
					<PlansSettings
						v-if="departureTabActive"
						:id="id"
						:staticPlan="staticPlan"
						:repeatingPlans="repeatingPlans"
						:effectiveLimitSoc="loadpoint?.effectiveLimitSoc"
						:effectivePlanTime="loadpoint?.effectivePlanTime ?? undefined"
						:effectivePlanSoc="loadpoint?.effectivePlanSoc"
						:effectivePlanStrategy="loadpoint?.effectivePlanStrategy"
						:planEnergy="loadpoint?.planEnergy"
						:limitEnergy="loadpoint?.limitEnergy"
						:socBasedPlanning="!!loadpoint?.socBasedPlanning"
						:socPerKwh="loadpoint?.socPerKwh"
						:rangePerSoc="loadpoint?.rangePerSoc"
						:smartCostType="smartCostType"
						:currency="currency"
						:mode="loadpoint?.mode"
						:capacity="vehicle?.capacity"
						:vehicle="vehicle"
						:vehicleLimitSoc="loadpoint?.vehicleLimitSoc"
						:planOverrun="loadpoint?.planOverrun"
						:forecast="forecast"
						@static-plan-updated="updateStaticPlan"
						@static-plan-removed="removeStaticPlan"
						@repeating-plans-updated="updateRepeatingPlans"
						@plan-strategy-updated="updatePlanStrategy"
					/>
					<Arrival
						v-if="arrivalTabActive"
						:id="id"
						:minSoc="vehicle?.minSoc"
						:limitSoc="vehicle?.limitSoc"
						:vehicleName="vehicle?.name"
						:vehicleNotReachable="loadpoint?.vehicleNotReachable"
						:socBasedCharging="loadpoint?.socBasedCharging"
						:rangePerSoc="loadpoint?.rangePerSoc"
						@minsoc-updated="setMinSoc"
						@limitsoc-updated="setLimitSoc"
					/>
				</div>
			</div>
		</GenericModal>
	</Teleport>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import PlansSettings from "./PlansSettings.vue";
import Arrival from "./Arrival.vue";
import api from "@/api";
import type {
	PlanStrategy,
	RepeatingPlan,
	StaticEnergyPlan,
	StaticPlan,
	StaticSocPlan,
} from "./types";
import type { CURRENCY, Forecast, SMART_COST_TYPE, UiLoadpoint, Vehicle } from "@/types/evcc";

export default defineComponent({
	name: "ChargingPlanModal",
	components: {
		GenericModal,
		PlansSettings,
		Arrival,
	},
	props: {
		loadpoints: { type: Array as PropType<UiLoadpoint[]>, default: () => [] },
		vehicles: { type: Array as PropType<Vehicle[]>, default: () => [] },
		smartCostType: String as PropType<SMART_COST_TYPE>,
		currency: String as PropType<CURRENCY>,
		forecast: Object as PropType<Forecast>,
	},
	data() {
		return {
			isModalVisible: false,
			activeTab: "departure",
			id: undefined as string | number | undefined,
		};
	},
	computed: {
		staticPlan(): StaticPlan | undefined {
			if (this.loadpoint?.socBasedPlanning) {
				const plan = this.vehicle?.plan as StaticSocPlan;
				if (plan) {
					return {
						soc: plan.soc,
						time: new Date(plan.time),
					};
				}
				return undefined;
			}
			if (this.loadpoint?.planEnergy && this.loadpoint?.planTime) {
				return {
					energy: this.loadpoint.planEnergy,
					time: new Date(this.loadpoint.planTime),
				};
			}
			return undefined;
		},
		loadpoint() {
			return this.loadpoints.find((loadpoint) => loadpoint.id === this.id);
		},
		vehicle() {
			return this.vehicles?.find((v) => v.name === this.loadpoint?.vehicleName);
		},
		modalTitle(): string {
			const baseTitle = this.$t("main.chargingPlan.modalTitle");
			if (this.loadpoint?.socBasedPlanning && this.vehicle) {
				return `${baseTitle}: ${this.vehicle.title}`;
			}
			return baseTitle;
		},
		departureTabActive(): boolean {
			return this.activeTab === "departure";
		},
		arrivalTabActive(): boolean {
			return this.activeTab === "arrival";
		},
		apiVehicle(): string {
			return `vehicles/${this.vehicle?.name}/`;
		},
		apiLoadpoint(): string {
			return `loadpoints/${this.id}/`;
		},
		repeatingPlans(): RepeatingPlan[] {
			if (
				this.vehicle &&
				this.vehicle.repeatingPlans &&
				this.vehicle.repeatingPlans.length > 0
			) {
				return this.vehicle.repeatingPlans || [];
			}
			return [];
		},
	},
	methods: {
		open(loadpointId?: string | number) {
			this.id = loadpointId;
			const modalRef = this.$refs["modal"] as InstanceType<typeof GenericModal> | undefined;
			modalRef?.open();
		},
		modalVisible(): void {
			this.isModalVisible = true;
		},
		modalInvisible(): void {
			this.isModalVisible = false;
		},
		setMinSoc(soc: number): void {
			api.post(`${this.apiVehicle}minsoc/${soc}`);
		},
		setLimitSoc(soc: number): void {
			api.post(`${this.apiVehicle}limitsoc/${soc}`);
		},
		updateStaticPlan(plan: StaticPlan): void {
			const timeISO = plan.time.toISOString();
			if (this.loadpoint?.socBasedPlanning) {
				const p = plan as StaticSocPlan;
				api.post(`${this.apiVehicle}plan/soc/${p.soc}/${timeISO}`, null);
			} else {
				const p = plan as StaticEnergyPlan;
				api.post(`${this.apiLoadpoint}plan/energy/${p.energy}/${timeISO}`, null);
			}
		},
		removeStaticPlan(): void {
			if (this.loadpoint?.socBasedPlanning) {
				api.delete(`${this.apiVehicle}plan/soc`);
			} else {
				api.delete(`${this.apiLoadpoint}plan/energy`);
			}
		},
		updateRepeatingPlans(plans: RepeatingPlan[]): void {
			api.post(`${this.apiVehicle}plan/repeating`, plans);
		},
		updatePlanStrategy(strategy: PlanStrategy): void {
			if (this.loadpoint?.socBasedPlanning) {
				api.post(`${this.apiVehicle}plan/strategy`, strategy);
			} else {
				api.post(`${this.apiLoadpoint}plan/strategy`, strategy);
			}
		},
		showDepartureTab(): void {
			this.activeTab = "departure";
		},
		showArrivalTab(): void {
			this.activeTab = "arrival";
		},
	},
});
</script>
