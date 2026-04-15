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
						:socBasedPlanning="!!socBasedPlanning"
						:socPerKwh="socPerKwh"
						:rangePerSoc="rangePerSoc"
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
						:vehicleNotReachable="vehicleNotReachable"
						:socBasedCharging="socBasedCharging"
						:rangePerSoc="rangePerSoc"
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
import { distanceValue } from "@/units";

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
			if (this.socBasedPlanning) {
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
		// TODO: refactor, see Vehicle.vue
		range() {
			return distanceValue(this.vehicleRange);
		},
		// TODO: refactor, see Vehicle.vue
		vehicleRange() {
			return this.loadpoint?.vehicleRange || 0;
		},
		// TODO: refactor, see Vehicle.vue
		vehicleSoc() {
			return this.loadpoint?.vehicleSoc || 0;
		},
		// TODO: refactor, see Vehicle.vue
		capacity() {
			return this.vehicle?.capacity || 0;
		},
		// TODO: refactor, see Vehicle.vue
		rangePerSoc() {
			if (this.vehicleSoc > 10 && this.range) {
				return Math.round((this.range / this.vehicleSoc) * 1e2) / 1e2;
			}
			return undefined;
		},
		// TODO: refactor, see Vehicle.vue
		socPerKwh() {
			if (this.capacity > 0) {
				return 100 / this.capacity;
			}
			return 0;
		},
		// TODO: refactor, see Loadpoint.vue
		vehicleNotReachable() {
			// online vehicle that was not reachable at startup
			const features = this.vehicle?.features || [];
			return features.includes("Offline") && features.includes("Retryable");
		},
		// TODO: refactor, see Loadpoint.vue
		vehicleKnown() {
			return !!this.loadpoint?.vehicleName;
		},
		// TODO: refactor, see Loadpoint.vue
		vehicleHasSoc() {
			return this.vehicleKnown && !this.vehicle?.features?.includes("Offline");
		},
		// TODO: refactor, see Loadpoint.vue
		socBasedCharging() {
			return this.vehicleHasSoc || (this.loadpoint && this.loadpoint?.vehicleSoc > 0);
		},
		// TODO: refactor, see Loadpoint.vue
		socBasedPlanning() {
			return this.socBasedCharging && this.vehicle?.capacity && this.vehicle?.capacity > 0;
		},
		loadpoint() {
			return this.loadpoints.find((loadpoint) => loadpoint.id === this.id);
		},
		vehicle() {
			return this.vehicles?.find((v) => v.name === this.loadpoint?.vehicleName);
		},
		modalTitle(): string {
			const baseTitle = this.$t("main.chargingPlan.modalTitle");
			if (this.socBasedPlanning && this.vehicle) {
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
			if (this.socBasedPlanning) {
				const p = plan as StaticSocPlan;
				api.post(`${this.apiVehicle}plan/soc/${p.soc}/${timeISO}`, null);
			} else {
				const p = plan as StaticEnergyPlan;
				api.post(`${this.apiLoadpoint}plan/energy/${p.energy}/${timeISO}`, null);
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
			api.post(`${this.apiVehicle}plan/repeating`, plans);
		},
		updatePlanStrategy(strategy: PlanStrategy): void {
			if (this.socBasedPlanning) {
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
