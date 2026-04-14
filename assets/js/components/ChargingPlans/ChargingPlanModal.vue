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
						v-bind="chargingPlansSettingsProps"
						@static-plan-updated="updateStaticPlan"
						@static-plan-removed="removeStaticPlan"
						@repeating-plans-updated="updateRepeatingPlans"
						@plan-strategy-updated="updatePlanStrategy"
					/>
					<Arrival
						v-if="arrivalTabActive"
						v-bind="chargingPlanArrival"
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
import collector from "@/mixins/collector";
import api from "@/api";
import type {
	PlanStrategy,
	RepeatingPlan,
	StaticEnergyPlan,
	StaticPlan,
	StaticSocPlan,
} from "./types";
import type { Vehicle } from "@/types/evcc";

export default defineComponent({
	name: "ChargingPlanModal",
	components: {
		GenericModal,
		PlansSettings,
		Arrival,
	},
	mixins: [collector],
	props: {
		socBasedPlanning: Boolean,
		vehicle: Object as PropType<Vehicle>,
	},
	data() {
		return {
			isModalVisible: false,
			activeTab: "departure",
			id: undefined as string | number | undefined,
		};
	},
	computed: {
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
		chargingPlansSettingsProps(): any {
			return this.collectProps(PlansSettings);
		},
		chargingPlanArrival(): any {
			return this.collectProps(Arrival);
		},
		apiVehicle(): string {
			return `vehicles/${this.vehicle?.name}/`;
		},
		apiLoadpoint(): string {
			return `loadpoints/${this.id}/`;
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
