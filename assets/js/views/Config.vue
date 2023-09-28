<template>
	<div class="container px-4">
		<TopHeader title="Configuration 🧪" />
		<div
			v-if="dirty"
			class="alert alert-secondary d-flex justify-content-between align-items-center my-4"
			role="alert"
		>
			<div><strong>Configuration changed.</strong> Please restart to see the effect.</div>
			<button
				type="button"
				class="btn btn-outline-secondary btn-sm"
				:disabled="restarting || offline"
				@click="restart"
			>
				<span
					v-if="restarting || offline"
					class="spinner-border spinner-border-sm"
					role="status"
					aria-hidden="true"
				></span>
				<span v-else> Restart </span>
			</button>
		</div>

		<div class="alert alert-danger my-4" role="alert">
			<strong>Highly experimental!</strong> Only play around with this settings if you know
			what your doing. Otherwise you might have to reset or manually repair you database.
		</div>

		<h2 class="my-4 mt-5">Home: <span>Zuhause</span></h2>
		<ul class="p-0 config-list mb-5">
			<DeviceCard
				:name="gridMeter?.config?.template || 'Grid meter'"
				:unconfigured="!gridMeter"
				:editable="!!gridMeter?.id"
				data-testid="grid"
				:tags="['-1.220 W', '3A 17A 11A', '72.128 kWh']"
				@configure="addMeter('grid')"
				@edit="editMeter(gridMeter.id, 'grid')"
			>
				<template #icon>
					<shopicon-regular-powersupply></shopicon-regular-powersupply>
				</template>
			</DeviceCard>
			<DeviceCard
				v-for="meter in pvMeters"
				:key="!!meter.name"
				:name="meter.config?.template || 'Solar system'"
				:editable="!!meter.id"
				data-testid="grid"
				:tags="['7.222 W']"
				@edit="editMeter(meter.id, 'pv')"
			>
				<template #icon>
					<shopicon-regular-sun></shopicon-regular-sun>
				</template>
			</DeviceCard>
			<DeviceCard
				v-for="meter in batteryMeters"
				:key="meter.name"
				:name="meter.config?.template || 'Battery storage'"
				:editable="!!meter.id"
				data-testid="grid"
				:tags="['220 W', '55%']"
				@edit="editMeter(meter.id, 'battery')"
			>
				<template #icon>
					<shopicon-regular-batterythreequarters></shopicon-regular-batterythreequarters>
				</template>
			</DeviceCard>
			<AddDeviceButton :title="$t('config.main.addPvBattery')" @add="addMeter" />
		</ul>

		<h2 class="my-4">Tariff</h2>

		<ul class="p-0 config-list mb-5">
			<DeviceCard name="Grid" editable data-testid="grid" @edit="todo">
				<template #icon>
					<shopicon-regular-powersupply></shopicon-regular-powersupply>
				</template>
			</DeviceCard>
			<DeviceCard name="Feed-in" unconfigured data-testid="grid" @edit="todo">
				<template #icon>
					<shopicon-regular-sun></shopicon-regular-sun>
				</template>
			</DeviceCard>
			<DeviceCard name="CO₂ estimate" unconfigured data-testid="grid" @edit="todo">
				<template #icon>
					<shopicon-regular-eco1></shopicon-regular-eco1>
				</template>
			</DeviceCard>
		</ul>

		<h2 class="my-4">Vehicles</h2>
		<div>
			<ul class="p-0 config-list mb-5">
				<DeviceCard
					v-for="vehicle in vehicles"
					:key="vehicle.id"
					:name="vehicle.config?.title || vehicle.name"
					:editable="vehicle.id >= 0"
					data-testid="vehicle"
					@edit="editVehicle(vehicle.id)"
				>
					<template #icon>
						<VehicleIcon :name="vehicle.config?.icon" />
					</template>
				</DeviceCard>
				<AddDeviceButton
					data-testid="add-vehicle"
					:title="$t('config.main.addVehicle')"
					@click="addVehicle"
				/>
			</ul>
		</div>
		<hr class="my-5" />
		<VehicleModal :id="selectedVehicleId" @vehicle-changed="vehicleChanged" />
		<MeterModal
			:id="selectedMeterId"
			:name="selectedMeterName"
			:type="selectedMeterType"
			@added="addMeterToSite"
			@updated="meterChanged"
			@removed="removeMeterFromSite"
		/>
	</div>
</template>

<script>
import TopHeader from "../components/TopHeader.vue";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import Modal from "bootstrap/js/dist/modal";
import api from "../api";
import VehicleIcon from "../components/VehicleIcon";
import VehicleModal from "../components/Config/VehicleModal.vue";
import DeviceCard from "../components/Config/DeviceCard.vue";
import AddDeviceButton from "../components/Config/AddDeviceButton.vue";
import MeterModal from "../components/Config/MeterModal.vue";

export default {
	name: "Config",
	components: { TopHeader, VehicleIcon, VehicleModal, DeviceCard, AddDeviceButton, MeterModal },
	props: {
		offline: Boolean,
	},
	data() {
		return {
			dirty: false,
			restarting: false,
			vehicles: [],
			meters: [],
			selectedVehicleId: undefined,
			selectedMeterId: undefined,
			selectedMeterType: undefined,
			site: { grid: "", pv: [], battery: [] },
		};
	},
	computed: {
		siteTitle() {
			return this.site?.title;
		},
		gridMeter() {
			const name = this.site?.grid;
			return this.getMetersByNames([name])[0];
		},
		pvMeters() {
			const names = this.site?.pv;
			return this.getMetersByNames(names);
		},
		batteryMeters() {
			const names = this.site?.battery;
			return this.getMetersByNames(names);
		},
		selectedMeterName() {
			return this.getMeterById(this.selectedMeterId)?.name;
		},
	},
	watch: {
		offline() {
			if (!this.offline) {
				this.restarting = false;
				this.loadAll();
			}
		},
	},
	mounted() {
		this.loadAll();
	},
	methods: {
		async loadAll() {
			this.loadVehicles();
			this.loadMeters();
			this.loadSite();
			this.loadDirty();
		},
		async loadDirty() {
			try {
				const response = await api.get("/config/dirty");
				this.dirty = response.data?.result;
			} catch (e) {
				console.error(e);
			}
		},
		async loadVehicles() {
			const response = await api.get("/config/devices/vehicle");
			this.vehicles = response.data?.result;
		},
		async loadMeters() {
			const response = await api.get("/config/devices/meter");
			this.meters = response.data?.result || [];
		},
		async loadSite() {
			const response = await api.get("/config/site");
			this.site = response.data?.result;
		},
		getMetersByNames(names) {
			if (!names || !this.meters) {
				return [];
			}
			return this.meters.filter((m) => names.includes(m.name));
		},
		getMeterById(id) {
			if (!id || !this.meters) {
				return undefined;
			}
			return this.meters.find((m) => m.id === id);
		},
		vehicleModal() {
			return Modal.getOrCreateInstance(document.getElementById("vehicleModal"));
		},
		meterModal() {
			return Modal.getOrCreateInstance(document.getElementById("meterModal"));
		},
		editMeter(id, type) {
			this.selectedMeterId = id;
			this.selectedMeterType = type;
			this.$nextTick(() => this.meterModal().show());
		},
		addMeter(type) {
			this.selectedMeterId = undefined;
			this.selectedMeterType = type;
			this.$nextTick(() => this.meterModal().show());
		},
		async meterChanged() {
			this.selectedMeterId = undefined;
			this.selectedMeterType = undefined;
			await this.loadMeters();
			this.loadDirty();
			this.meterModal().hide();
		},
		editVehicle(id) {
			this.selectedVehicleId = id;
			this.$nextTick(() => this.vehicleModal().show());
		},
		addVehicle() {
			this.selectedVehicleId = undefined;
			this.$nextTick(() => this.vehicleModal().show());
		},
		vehicleChanged() {
			this.selectedVehicleId = undefined;
			this.vehicleModal().hide();
			this.loadVehicles();
			this.loadDirty();
		},
		addMeterToSite(type, name) {
			if (type === "grid") {
				this.site.grid = name;
			} else {
				this.site[type].push(name);
			}
			this.saveSite(type);
		},
		removeMeterFromSite(type, name) {
			if (type === "grid") {
				this.site.grid = "";
			} else {
				this.site[type] = this.site[type].filter((i) => i !== name);
			}
			this.saveSite(type);
		},
		async saveSite(key) {
			const body = key ? { [key]: this.site[key] } : this.site;
			await api.put("/config/site", body);
			this.loadDirty();
			this.loadSite();
		},
		todo() {
			alert("not implemented yet");
		},
		async restart() {
			try {
				await api.post("shutdown");
				this.restarting = true;
			} catch (e) {
				alert("Unabled to restart server.");
			}
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
