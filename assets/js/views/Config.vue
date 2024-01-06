<template>
	<div class="container px-4">
		<header class="d-flex justify-content-between align-items-center py-3 mb-4">
			<h1 class="mb-1 pt-1 d-flex text-nowrap">
				<router-link class="dropdown-item mx-2 me-2" to="/">
					<shopicon-bold-arrowback size="s" class="back"></shopicon-bold-arrowback>
				</router-link>
				Configuration 🧪
			</h1>
			<TopNavigation />
		</header>

		<div class="alert alert-danger mb-5" role="alert">
			<strong>Highly experimental!</strong> Only play around with this settings if you know
			what your doing. Otherwise you might have to reset or manually repair you database.
		</div>

		<!--
		<h2 class="d-flex align-items-center text-evcc">
			<shopicon-regular-powersupply size="m" class="me-2"></shopicon-regular-powersupply>
			Grid
		</h2>
		<div>
			<ul class="p-0">
				<li class="d-flex align-items-center">
					Grid meter <small class="mx-2">E3/DC</small>

					<button class="btn btn-sm btn-link text-gray" @click="todo">edit</button>
				</li>
				<li class="d-flex align-items-center">
					Grid use <small class="ms-2">32ct/kWh fixed</small>
					<button class="btn btn-sm btn-link text-gray" @click="todo">edit</button>
				</li>
				<li class="d-flex align-items-center">
					Grid feed-in <small class="ms-2">8ct/kWh fixed</small>
					<button class="btn btn-sm btn-link text-gray" @click="todo">edit</button>
				</li>
				<li class="d-flex align-items-center">
					Grid CO₂ <small class="ms-2">none</small>
					<button class="btn btn-sm btn-link text-gray" @click="todo">configure</button>
				</li>
			</ul>
		</div>
		<hr />

		<h2 class="d-flex align-items-center text-evcc">
			<shopicon-regular-sun size="m" class="me-2"></shopicon-regular-sun>
			Photovoltaics
		</h2>
		<div>
			<ul class="p-0">
				<li v-for="(meter, index) in pvs" :key="index" class="d-flex align-items-center">
					PV {{ index + 1 }}
					<shopicon-regular-warning
						v-if="index === 3"
						size="s"
						class="text-warning ms-2"
					></shopicon-regular-warning>
					<button class="btn btn-sm btn-link text-gray" @click="todo">edit</button>
				</li>
				<li class="d-flex align-items-center">
					<button class="btn btn-sm btn-link text-gray px-0" @click="todo">
						+ add PV
					</button>
				</li>
			</ul>
		</div>
		<hr />

		<h2 class="d-flex align-items-center text-evcc">
			<shopicon-regular-batterythreequarters
				size="m"
				class="me-2"
			></shopicon-regular-batterythreequarters>
			Battery storage
		</h2>
		<div>
			<ul class="p-0">
				<li
					v-for="(meter, index) in batteries"
					:key="index"
					class="d-flex align-items-center"
				>
					Battery {{ index + 1 }}
					<button class="btn btn-sm btn-link text-gray" @click="todo">edit</button>
				</li>
				<li class="d-flex align-items-center">
					<button class="btn btn-sm btn-link text-gray px-0" @click="todo">
						+ add battery
					</button>
				</li>
			</ul>
		</div>
		<hr />

		<h2 class="d-flex align-items-center text-evcc">
			<shopicon-regular-lightning size="m" class="me-2"></shopicon-regular-lightning>
			Loadpoints
		</h2>
		<div>Bar</div>
		<hr />
		-->

		<h2 class="my-4">My Vehicles</h2>
		<div>
			<ul class="p-0 config-list">
				<li
					v-for="(vehicle, index) in vehicles"
					:key="index"
					class="config-entry"
					data-testid="vehicle"
				>
					<div class="d-flex align-items-center mb-2">
						<VehicleIcon :name="vehicle.config?.icon" class="me-2" />
						<span class="flex-grow-1 text-nowrap text-truncate">{{
							vehicle.config?.title || vehicle.name
						}}</span>
						<button
							v-if="vehicle.id"
							class="btn btn-sm btn-link text-gray"
							@click="editVehicle(vehicle.id)"
						>
							{{ $t("config.main.edit") }}
						</button>
						<span v-else class="text-gray opacity-50 px-2 py-1" disabled>yaml</span>
					</div>
					<div v-if="false" class="d-flex mb-2" @click="todo">
						<!-- example for later -->
						<span
							v-if="index > 1 && index < 6"
							title="Charge"
							class="badge text-bg-secondary me-1 mb-1"
							>33%</span
						>
						<span
							v-if="index > 2"
							title="Capacity"
							class="badge text-bg-secondary me-1 mb-1"
							>42 kWh</span
						>
						<span
							v-if="index < 4"
							title="Power"
							class="badge text-bg-secondary me-1 mb-1"
							>+4,3 kW</span
						>
						<span title="Odometer" class="badge text-bg-secondary me-1 mb-1"
							>23.123 km</span
						>
						<span
							v-if="index == 2"
							title="Climate"
							class="badge text-bg-secondary me-1 mb-1"
							>climate</span
						>
					</div>
				</li>
			</ul>
			<button class="btn btn-outline-secondary" @click="addVehicle">
				{{ $t("config.main.addVehicle") }}
			</button>
			<VehicleModal :id="vehicleId" @vehicle-changed="vehicleChanged" />
		</div>
		<hr class="my-5" />

		<!--
		<h2 class="d-flex align-items-center text-evcc">
			<shopicon-regular-megaphone size="m" class="me-2"></shopicon-regular-megaphone>
			Notifications
		</h2>
		<div>Bar</div>
		<hr />
	-->
	</div>
</template>

<script>
import TopNavigation from "../components/TopNavigation.vue";
import "@h2d2/shopicons/es/bold/arrowback";
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/warning";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/megaphone";
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import "@h2d2/shopicons/es/regular/home";
import Modal from "bootstrap/js/dist/modal";
import api from "../api";
import VehicleIcon from "../components/VehicleIcon";
import VehicleModal from "../components/Config/VehicleModal.vue";

export default {
	name: "Config",
	components: { TopNavigation, VehicleIcon, VehicleModal },
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
			this.vehicleModal().show();
		},
		addVehicle() {
			this.vehicleId = undefined;
			this.vehicleModal().show();
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
.back {
	width: 22px;
	height: 22px;
	position: relative;
	top: -2px;
}
.container {
	max-width: 700px;
}
.config-list {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
	grid-gap: 1rem;
}
.config-entry {
	border-radius: 1rem;
	color: var(--evcc-default-text);
	background: var(--evcc-box);
	padding: 1rem 1rem 0.5rem;
	display: block;
	list-style-type: none;
}
</style>
