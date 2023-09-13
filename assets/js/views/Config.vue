<template>
	<div class="container px-4">
		<TopHeader title="Configuration 🧪" />

		<div class="alert alert-danger mb-5" role="alert">
			<strong>Highly experimental!</strong> Only play around with this settings if you know
			what your doing. Otherwise you might have to reset or manually repair you database.
		</div>

		<h2 class="my-4">Vehicles</h2>
		<div>
			<ul class="p-0 config-list">
				<DeviceCard
					v-for="vehicle in vehicles"
					:key="vehicle.id"
					:vehicle="vehicle"
					:name="vehicle.config?.title || vehicle.name"
					:editable="vehicle.id >= 0"
					data-testid="vehicle"
					@edit="editVehicle(vehicle.id)"
				>
					<template #icon>
						<VehicleIcon :name="vehicle.config?.icon" />
					</template>
				</DeviceCard>
			</ul>
			<button class="btn btn-outline-secondary" @click="addVehicle">
				{{ $t("config.main.addVehicle") }}
			</button>
			<VehicleModal :id="vehicleId" @vehicle-changed="vehicleChanged" />
		</div>
		<hr class="my-5" />
	</div>
</template>

<script>
import TopHeader from "../components/TopHeader.vue";
import "@h2d2/shopicons/es/bold/arrowback";
import Modal from "bootstrap/js/dist/modal";
import api from "../api";
import VehicleIcon from "../components/VehicleIcon";
import VehicleModal from "../components/Config/VehicleModal.vue";
import DeviceCard from "../components/Config/DeviceCard.vue";

export default {
	name: "Config",
	components: { TopHeader, VehicleIcon, VehicleModal, DeviceCard },
	data() {
		return {
			vehicles: [],
			loadpoints: [],
			pvs: [],
			batteries: [],
			grid: null,
			vehicleId: undefined,
		};
	},
	mounted() {
		this.loadVehicles();
		this.loadLoadpoints();
		this.loadMeters();
	},
	methods: {
		async loadVehicles() {
			const response = await api.get("/config/devices/vehicle");
			this.vehicles = response.data?.result;
		},
		async loadLoadpoints() {
			// TODO: add GET loadpoints API
			const response = await api.get("/config/devices/charger");
			this.loadpoints = response.data?.result;
		},
		async loadMeters() {
			// TODO: split this into separate endpoints for pv, battery, grid
			const response = await api.get("/config/devices/meter");
			const meters = response.data?.result || [];
			this.pvs = meters;
			this.batteries = meters;
			this.grid = meters[0];
		},
		vehicleModal() {
			return Modal.getOrCreateInstance(document.getElementById("vehicleModal"));
		},
		editVehicle(id) {
			this.vehicleId = id;
			this.$nextTick(() => this.vehicleModal().show());
		},
		addVehicle() {
			this.vehicleId = undefined;
			this.$nextTick(() => this.vehicleModal().show());
		},
		vehicleChanged() {
			this.vehicleId = undefined;
			this.vehicleModal().hide();
			this.loadVehicles();
		},
		todo() {
			alert("not implemented yet");
		},
	},
};
</script>
<style scoped>
.container {
	max-width: 700px;
}
.config-list {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
	grid-gap: 1rem;
}
</style>
