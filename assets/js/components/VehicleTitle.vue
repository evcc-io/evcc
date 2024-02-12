<template>
	<div class="d-flex justify-content-between mb-3 align-items-center" data-testid="vehicle-title">
		<h4 class="d-flex align-items-center m-0 flex-grow-1 overflow-hidden">
			<shopicon-regular-refresh
				v-if="iconType === 'refresh'"
				ref="refresh"
				data-bs-toggle="tooltip"
				:title="$t('main.vehicle.detectionActive')"
				class="me-2 flex-shrink-0 spin"
			></shopicon-regular-refresh>
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
				:vehicles="otherVehicles"
				@change-vehicle="changeVehicle"
				@remove-vehicle="removeVehicle"
			>
				<span class="flex-grow-1 text-truncate vehicle-name">
					{{ name }}
				</span>
			</VehicleOptions>
			<span v-else class="flex-grow-1 text-truncate vehicle-name">
				{{ name }}
			</span>
		</h4>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/refresh";
import "@h2d2/shopicons/es/regular/cablecharge";
import Tooltip from "bootstrap/js/dist/tooltip";
import VehicleIcon from "./VehicleIcon";
import VehicleOptions from "./VehicleOptions.vue";
import collector from "../mixins/collector";

export default {
	name: "VehicleTitle",
	components: { VehicleOptions, VehicleIcon },
	mixins: [collector],
	props: {
		connected: Boolean,
		id: [String, Number],
		vehicleDetectionActive: Boolean,
		icon: String,
		vehicleName: String,
		vehicles: { type: Array, default: () => [] },
		title: String,
	},
	emits: ["change-vehicle", "remove-vehicle"],
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
		vehicleKnown() {
			return !!this.vehicleName;
		},
		otherVehicles() {
			return this.vehicles.filter((v) => v.name !== this.vehicleName);
		},
		showOptions() {
			return this.vehicleKnown || this.vehicles.length;
		},
		vehicleOptionsProps: function () {
			return this.collectProps(VehicleOptions);
		},
	},
	watch: {
		iconType: function () {
			this.tooltip();
		},
	},
	mounted: function () {
		this.tooltip();
	},
	methods: {
		changeVehicle(name) {
			this.$emit("change-vehicle", name);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
		tooltip() {
			this.$nextTick(() => {
				if (this.$refs.refresh) {
					new Tooltip(this.$refs.refresh);
				}
			});
		},
	},
};
</script>

<style scoped>
.options {
	margin-right: -0.75rem;
}
.vehicle-name {
	text-decoration-color: var(--evcc-gray);
}
.options .vehicle-name {
	text-decoration: underline;
}
.spin {
	animation: rotation 1s infinite cubic-bezier(0.37, 0, 0.63, 1);
}
.spin :deep(svg) {
	/* workaround to fix the not perfectly centered shopicon. Remove once its fixed in @h2d2/shopicons */
	transform: translateY(-0.7px);
}
@keyframes rotation {
	from {
		transform: rotate(0deg);
	}
	to {
		transform: rotate(360deg);
	}
}
</style>
