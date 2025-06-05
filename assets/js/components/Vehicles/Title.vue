<template>
	<div class="d-flex justify-content-between mb-3 align-items-center" data-testid="vehicle-title">
		<h4 class="d-flex align-items-center m-0 flex-grow-1 overflow-hidden">
			<div
				v-if="iconType === 'refresh'"
				ref="refresh"
				class="me-2 flex-shrink-0 spin"
				:title="$t('main.vehicle.detectionActive')"
				data-bs-toggle="tooltip"
			>
				<Sync />
			</div>
			<VehicleIcon
				v-else-if="iconType === 'vehicle'"
				:name="icon"
				class="me-2 flex-shrink-0"
			/>
			<shopicon-regular-cablecharge
				v-else
				class="me-2 flex-shrink-0"
			></shopicon-regular-cablecharge>
			<VehicleOptions
				v-if="showOptions"
				v-bind="vehicleOptionsProps"
				:id="id"
				class="options"
				:selected="vehicleName"
				@change-vehicle="changeVehicle"
				@remove-vehicle="removeVehicle"
			>
				<span class="flex-grow-1 text-truncate vehicle-name" data-testid="vehicle-name">
					{{ name }}
				</span>
			</VehicleOptions>
			<span v-else class="flex-grow-1 text-truncate vehicle-name" data-testid="vehicle-name">
				{{ name }}
			</span>
			<button
				v-if="vehicleNotReachable"
				ref="notReachable"
				class="ms-2 btn-neutral"
				data-bs-toggle="tooltip"
				:title="$t('main.vehicle.notReachable')"
				type="button"
				data-testid="vehicle-not-reachable-icon"
				@click="openHelpModal"
			>
				<CloudOffline class="evcc-gray" />
			</button>
		</h4>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/cablecharge";
import Tooltip from "bootstrap/js/dist/tooltip";
import Modal from "bootstrap/js/dist/modal";
import VehicleIcon from "../VehicleIcon";
import Options from "./Options.vue";
import CloudOffline from "../MaterialIcon/CloudOffline.vue";
import Sync from "../MaterialIcon/Sync.vue";
import collector from "@/mixins/collector";
import { defineComponent, type PropType } from "vue";
import type { SelectOption, Vehicle } from "@/types/evcc";

export default defineComponent({
	name: "VehicleTitle",
	components: { VehicleOptions: Options, VehicleIcon, Sync, CloudOffline },
	mixins: [collector],
	props: {
		connected: Boolean,
		id: [String, Number],
		vehicleDetectionActive: Boolean,
		vehicleNotReachable: Boolean,
		icon: String,
		vehicleName: String,
		vehicles: { type: Array as PropType<Vehicle[]>, default: () => [] },
		title: String,
	},
	emits: ["change-vehicle", "remove-vehicle"],
	data() {
		return {
			refreshTooltip: null as Tooltip | null,
			notReachableTooltip: null as Tooltip | null,
		};
	},
	computed: {
		iconType() {
			if (this.vehicleDetectionActive) {
				return "refresh";
			}
			if (this.connected) {
				return "vehicle";
			}
			return null;
		},
		name() {
			if (this.title) {
				return this.title;
			}
			if (this.connected) {
				return this.$t("main.vehicle.unknown");
			}
			return this.$t("main.vehicle.none");
		},
		vehicleOptions(): SelectOption<string>[] {
			return this.vehicles.map((v) => ({
				name: v.name,
				value: v.title,
			}));
		},
		vehicleKnown() {
			return !!this.vehicleName;
		},
		showOptions() {
			return this.vehicleKnown || this.vehicles.length;
		},
		vehicleOptionsProps() {
			return this.collectProps(Options);
		},
	},
	watch: {
		iconType() {
			this.initTooltip();
		},
	},
	mounted() {
		this.initTooltip();
	},
	methods: {
		changeVehicle(name: string) {
			this.$emit("change-vehicle", name);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
		initTooltip() {
			this.$nextTick(() => {
				this.refreshTooltip?.dispose();
				this.notReachableTooltip?.dispose();
				if (this.$refs["refresh"]) {
					this.refreshTooltip = new Tooltip(this.$refs["refresh"]);
				}
				if (this.$refs["notReachable"]) {
					this.notReachableTooltip = new Tooltip(this.$refs["notReachable"]);
				}
			});
		},
		openHelpModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("helpModal") as HTMLElement
			);
			modal.show();
			this.initTooltip();
		},
	},
});
</script>

<style scoped>
.vehicle-name {
	text-decoration-color: var(--evcc-gray);
}
.options .vehicle-name {
	text-decoration: underline;
}
.spin {
	animation: rotation 1s infinite cubic-bezier(0.37, 0, 0.63, 1);
}
@keyframes rotation {
	from {
		transform: rotate(0deg);
	}
	to {
		transform: rotate(-360deg);
	}
}
</style>
