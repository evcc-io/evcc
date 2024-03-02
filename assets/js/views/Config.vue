<template>
	<div class="root safe-area-inset">
		<div class="container px-4">
			<TopHeader title="Configuration 🧪" />
			<div class="wrapper">
				<div
					v-if="dirty"
					class="alert alert-secondary d-flex justify-content-between align-items-center my-4"
					role="alert"
				>
					<div v-if="restarting"><strong>Restarting evcc.</strong> Please wait ...</div>
					<div v-else>
						<strong>Configuration changed.</strong> Please restart to see the effect.
					</div>
					<button
						type="button"
						class="btn btn-outline-dark btn-sm"
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
					<strong>Highly experimental!</strong> Only play around with these settings if you
					know what you're doing. Otherwise you might have to reset or manually repair your
					database.
				</div>

				<h2 class="my-4 mt-5">General</h2>
				<SiteSettings @site-changed="siteChanged" />

				<h2 class="my-4 mt-5">Grid, PV & Battery Systems</h2>
				<ul class="p-0 config-list">
					<DeviceCard
						:name="gridMeter?.config?.template || 'Grid meter'"
						:unconfigured="!gridMeter"
						:editable="!!gridMeter?.id"
						data-testid="grid"
						@configure="addMeter('grid')"
						@edit="editMeter(gridMeter.id, 'grid')"
					>
						<template #icon>
							<shopicon-regular-powersupply></shopicon-regular-powersupply>
						</template>
						<template #tags>
							<DeviceTags :tags="deviceTags('meter', gridMeter?.name)" />
						</template>
					</DeviceCard>
					<DeviceCard
						v-for="meter in pvMeters"
						:key="!!meter.name"
						:name="meter.config?.template || 'Solar system'"
						:editable="!!meter.id"
						data-testid="pv"
						@edit="editMeter(meter.id, 'pv')"
					>
						<template #icon>
							<shopicon-regular-sun></shopicon-regular-sun>
						</template>
						<template #tags>
							<DeviceTags :tags="deviceTags('meter', meter.name)" />
						</template>
					</DeviceCard>
					<DeviceCard
						v-for="meter in batteryMeters"
						:key="meter.name"
						:name="meter.config?.template || 'Battery storage'"
						:editable="!!meter.id"
						data-testid="battery"
						@edit="editMeter(meter.id, 'battery')"
					>
						<template #icon>
							<shopicon-regular-batterythreequarters></shopicon-regular-batterythreequarters>
						</template>
						<template #tags>
							<DeviceTags :tags="deviceTags('meter', meter.name)" />
						</template>
					</DeviceCard>
					<AddDeviceButton :title="$t('config.main.addPvBattery')" @add="addMeter" />
				</ul>

				<h2 class="my-4 wip">Tariffs</h2>

				<ul class="p-0 config-list wip">
					<DeviceCard
						name="Grid"
						unconfigured
						data-testid="tariff-grid"
						@configure="todo"
					>
						<template #icon>
							<shopicon-regular-money></shopicon-regular-money>
						</template>
					</DeviceCard>
					<DeviceCard
						name="Feed-in"
						unconfigured
						data-testid="tariff-feedin"
						@configure="todo"
					>
						<template #icon>
							<shopicon-regular-receivepayment></shopicon-regular-receivepayment>
						</template>
					</DeviceCard>
					<DeviceCard
						name="CO₂ estimate"
						unconfigured
						data-testid="tariff-co2"
						@configure="todo"
					>
						<template #icon>
							<shopicon-regular-eco1></shopicon-regular-eco1>
						</template>
					</DeviceCard>
				</ul>

				<h2 class="my-4 wip">Charge Points</h2>

				<ul class="p-0 config-list wip">
					<DeviceCard
						name="Fake Carport"
						editable
						data-testid="chargepoint-1"
						@edit="todo"
					>
						<template #icon>
							<shopicon-regular-cablecharge></shopicon-regular-cablecharge>
						</template>
						<template #tags>
							<DeviceTags :tags="{ power: 0 }" />
						</template>
					</DeviceCard>
					<AddDeviceButton
						data-testid="add-loadpoint"
						:title="$t('config.main.addLoadpoint')"
						@click="todo"
					/>
				</ul>

				<h2 class="my-4">Vehicles</h2>
				<div>
					<ul class="p-0 config-list">
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
							<template #tags>
								<DeviceTags :tags="deviceTags('vehicle', vehicle.name)" />
							</template>
						</DeviceCard>
						<AddDeviceButton
							data-testid="add-vehicle"
							:title="$t('config.main.addVehicle')"
							@click="addVehicle"
						/>
					</ul>
				</div>
				<h2 class="my-4 wip">Integrations</h2>

				<ul class="p-0 config-list wip">
					<DeviceCard name="MQTT" unconfigured data-testid="mqtt" @configure="todo">
						<template #icon>
							<shopicon-regular-fastdelivery1></shopicon-regular-fastdelivery1>
						</template>
					</DeviceCard>
					<DeviceCard
						name="Notifications"
						unconfigured
						data-testid="eebus"
						@configure="todo"
					>
						<template #icon>
							<shopicon-regular-sendit></shopicon-regular-sendit>
						</template>
					</DeviceCard>
					<DeviceCard name="InfluxDB" unconfigured data-testid="influx" @configure="todo">
						<template #icon>
							<shopicon-regular-diagram></shopicon-regular-diagram>
						</template>
					</DeviceCard>
					<DeviceCard name="EEBus" unconfigured data-testid="eebus" @configure="todo">
						<template #icon>
							<shopicon-regular-polygon></shopicon-regular-polygon>
						</template>
					</DeviceCard>
				</ul>

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
		</div>
	</div>
</template>

<script>
import TopHeader from "../components/TopHeader.vue";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/money";
import "@h2d2/shopicons/es/regular/receivepayment";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/fastdelivery1";
import "@h2d2/shopicons/es/regular/sendit";
import "@h2d2/shopicons/es/regular/diagram";
import "@h2d2/shopicons/es/regular/polygon";
import "@h2d2/shopicons/es/regular/cablecharge";
import Modal from "bootstrap/js/dist/modal";
import api from "../api";
import VehicleIcon from "../components/VehicleIcon";
import VehicleModal from "../components/Config/VehicleModal.vue";
import DeviceCard from "../components/Config/DeviceCard.vue";
import DeviceTags from "../components/Config/DeviceTags.vue";
import AddDeviceButton from "../components/Config/AddDeviceButton.vue";
import MeterModal from "../components/Config/MeterModal.vue";
import SiteSettings from "../components/Config/SiteSettings.vue";
import formatter from "../mixins/formatter";

export default {
	name: "Config",
	components: {
		TopHeader,
		SiteSettings,
		VehicleIcon,
		VehicleModal,
		DeviceCard,
		DeviceTags,
		AddDeviceButton,
		MeterModal,
	},
	props: {
		offline: Boolean,
		notifications: Array,
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
			deviceValueTimeout: undefined,
			deviceValues: {},
		};
	},
	mixins: [formatter],
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
	unmounted() {
		clearTimeout(this.deviceValueTimeout);
	},
	methods: {
		async loadAll() {
			await this.loadVehicles();
			await this.loadMeters();
			await this.loadSite();
			await this.loadDirty();
			await this.updateValues();
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
			this.vehicles = response.data?.result || [];
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
			this.meterModal().hide();
			await this.loadDirty();
			await this.updateValues();
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
		siteChanged() {
			this.loadDirty();
		},
		addMeterToSite(type, name) {
			if (type === "grid") {
				this.site.grid = name;
			} else {
				if (!this.site[type]) {
					this.site[type] = [];
				}
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
			await this.loadSite();
			await this.loadDirty();
			await this.updateValues();
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
		async updateDeviceValue(type, name) {
			try {
				const response = await api.get(`/config/devices/${type}/${name}/status`);
				if (!this.deviceValues[type]) this.deviceValues[type] = {};
				this.deviceValues[type][name] = response.data.result;
			} catch (error) {
				console.error("Error fetching device values for", type, name, error);
				return null;
			}
		},
		async updateValues() {
			clearTimeout(this.deviceValueTimeout);

			const promises = [
				...this.meters.map((meter) => this.updateDeviceValue("meter", meter.name)),
				...this.vehicles.map((vehicle) => this.updateDeviceValue("vehicle", vehicle.name)),
			];

			await Promise.all(promises);

			this.deviceValueTimeout = setTimeout(this.updateValues, 10000);
		},
		deviceTags(type, id) {
			return this.deviceValues[type]?.[id] || [];
		},
	},
};
</script>
<style scoped>
.config-list {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
	grid-gap: 1rem;
	margin-bottom: 5rem;
}
.wrapper {
	max-width: 900px;
	margin: 0 auto;
}
.wip {
	opacity: 0.2 !important;
}
</style>
