<template>
	<div class="root safe-area-inset">
		<div class="container px-4">
			<TopHeader :title="$t('config.main.title')" />
			<div class="wrapper pb-5">
				<h2 class="my-4 mt-5">{{ $t("config.section.general") }}</h2>
				<GeneralConfig @site-changed="siteChanged" />

				<div v-if="$hiddenFeatures()">
					<!--
					<hr class="my-5" />

					<div class="alert alert-danger my-4" role="alert">
						<strong>Highly experimental!</strong> Only play around with these settings
						if you know what you're doing. Otherwise you might have to reset or manually
						repair your database.
					</div>
-->
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
							name="Tariffs"
							editable
							data-testid="tariffs"
							@edit="openModal('tariffsModal')"
						>
							<template #icon>
								<shopicon-regular-receivepayment></shopicon-regular-receivepayment>
							</template>
							<template #tags>
								<DeviceTags :tags="tariffTags" />
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

						<h2 class="my-4 mt-5">Integrations</h2>

						<ul class="p-0 config-list">
							<DeviceCard
								:name="$t('config.mqtt.title')"
								editable
								data-testid="mqtt"
								@configure="openModal('mqttModal')"
								@edit="openModal('mqttModal')"
							>
								<template #icon><MqttIcon /></template>
							</DeviceCard>
							<DeviceCard
								:name="$t('config.messaging.title')"
								editable
								data-testid="messaging"
								@edit="openModal('messagingModal')"
							>
								<template #icon><NotificationIcon /></template>
							</DeviceCard>
							<DeviceCard
								:name="$t('config.influx.title')"
								editable
								data-testid="influx"
								@edit="openModal('influxModal')"
							>
								<template #icon><InfluxIcon /></template>
							</DeviceCard>
							<DeviceCard
								:name="$t('config.eebus.title')"
								editable
								data-testid="eebus"
								@edit="openModal('eebusModal')"
							>
								<template #icon><EebusIcon /></template>
							</DeviceCard>
							<DeviceCard
								:name="$t('config.circuits.title')"
								editable
								data-testid="circuits"
								@edit="openModal('circuitsModal')"
							>
								<template #icon><CircuitsIcon /></template>
							</DeviceCard>
							<DeviceCard
								:name="$t('config.modbusproxy.title')"
								editable
								data-testid="modbusproxy"
								@edit="openModal('modbusProxyModal')"
							>
								<template #icon><ModbusProxyIcon /></template>
							</DeviceCard>
							<DeviceCard
								:name="$t('config.hems.title')"
								editable
								data-testid="hems"
								@edit="openModal('hemsModal')"
							>
								<template #icon><HemsIcon /></template>
							</DeviceCard>
						</ul>
					</div>
				</div>

				<hr class="my-5" />

				<h2 class="my-4 mt-5">{{ $t("config.section.system") }}</h2>
				<div class="round-box p-4 d-flex gap-4 mb-5">
					<router-link to="/log" class="btn btn-outline-secondary">
						{{ $t("config.system.logs") }}
					</router-link>
					<button class="btn btn-outline-danger" @click="restart">
						{{ $t("config.system.restart") }}
					</button>
				</div>

				<VehicleModal :id="selectedVehicleId" @vehicle-changed="vehicleChanged" />
				<MeterModal
					:id="selectedMeterId"
					:name="selectedMeterName"
					:type="selectedMeterType"
					@added="addMeterToSite"
					@updated="meterChanged"
					@removed="removeMeterFromSite"
				/>
				<EebusModal @changed="loadDirty" />
				<HemsModal @changed="loadDirty" />
				<InfluxModal @changed="loadDirty" />
				<MessagingModal @changed="loadDirty" />
				<ModbusProxyModal @changed="loadDirty" />
				<CircuitsModal @changed="loadDirty" />
				<MqttModal @changed="loadDirty" />
				<TariffsModal @changed="loadDirty" />
				<NetworkModal @changed="loadDirty" />
			</div>
		</div>
	</div>
</template>

<script>
import TopHeader from "../components/TopHeader.vue";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/receivepayment";
import "@h2d2/shopicons/es/regular/cablecharge";
import AddDeviceButton from "../components/Config/AddDeviceButton.vue";
import api from "../api";
import CircuitsIcon from "../components/MaterialIcon/Circuits.vue";
import CircuitsModal from "../components/Config/CircuitsModal.vue";
import collector from "../mixins/collector";
import DeviceCard from "../components/Config/DeviceCard.vue";
import DeviceTags from "../components/Config/DeviceTags.vue";
import EebusIcon from "../components/MaterialIcon/Eebus.vue";
import EebusModal from "../components/Config/EebusModal.vue";
import formatter from "../mixins/formatter";
import GeneralConfig from "../components/Config/GeneralConfig.vue";
import HemsIcon from "../components/MaterialIcon/Hems.vue";
import HemsModal from "../components/Config/HemsModal.vue";
import InfluxIcon from "../components/MaterialIcon/Influx.vue";
import InfluxModal from "../components/Config/InfluxModal.vue";
import MessagingModal from "../components/Config/MessagingModal.vue";
import MeterModal from "../components/Config/MeterModal.vue";
import Modal from "bootstrap/js/dist/modal";
import ModbusProxyIcon from "../components/MaterialIcon/ModbusProxy.vue";
import ModbusProxyModal from "../components/Config/ModbusProxyModal.vue";
import MqttIcon from "../components/MaterialIcon/Mqtt.vue";
import MqttModal from "../components/Config/MqttModal.vue";
import NetworkModal from "../components/Config/NetworkModal.vue";
import NotificationIcon from "../components/MaterialIcon/Notification.vue";
import restart, { performRestart } from "../restart";
import store from "../store";
import TariffsModal from "../components/Config/TariffsModal.vue";
import VehicleIcon from "../components/VehicleIcon";
import VehicleModal from "../components/Config/VehicleModal.vue";

export default {
	name: "Config",
	components: {
		AddDeviceButton,
		CircuitsIcon,
		CircuitsModal,
		DeviceCard,
		DeviceTags,
		EebusIcon,
		EebusModal,
		GeneralConfig,
		HemsIcon,
		HemsModal,
		InfluxIcon,
		InfluxModal,
		MessagingModal,
		MeterModal,
		ModbusProxyIcon,
		ModbusProxyModal,
		MqttIcon,
		MqttModal,
		NetworkModal,
		NotificationIcon,
		TariffsModal,
		TopHeader,
		VehicleIcon,
		VehicleModal,
	},
	props: {
		offline: Boolean,
		notifications: Array,
	},
	data() {
		return {
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
	mixins: [formatter, collector],
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
		tariffTags() {
			const { currency, tariffGrid, tariffFeedIn, tariffCo2 } = store.state;
			const tags = {};
			if (currency) {
				tags.currency = { value: currency };
			}
			if (tariffGrid) {
				tags.gridPrice = { value: tariffGrid, options: { currency } };
			}
			if (tariffFeedIn) {
				tags.feedinPrice = { value: tariffFeedIn * -1, options: { currency } };
			}
			if (tariffCo2) {
				tags.co2 = { value: tariffCo2 };
			}
			return tags;
		},
	},
	watch: {
		offline() {
			if (!this.offline) {
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
			const response = await api.get("/config/dirty");
			if (response.data?.result) {
				restart.restartNeeded = true;
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
			await performRestart();
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
			if (!this.offline) {
				const promises = [
					...this.meters.map((meter) => this.updateDeviceValue("meter", meter.name)),
					...this.vehicles.map((vehicle) =>
						this.updateDeviceValue("vehicle", vehicle.name)
					),
				];

				await Promise.all(promises);
			}
			this.deviceValueTimeout = setTimeout(this.updateValues, 10000);
		},
		deviceTags(type, id) {
			return this.deviceValues[type]?.[id] || [];
		},
		openModal(id) {
			const $el = document.getElementById(id);
			if ($el) {
				Modal.getOrCreateInstance($el).show();
			} else {
				console.error(`modal ${id} not found`);
			}
		},
	},
};
</script>
<style scoped>
.config-list {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
	grid-gap: 2rem;
	margin-bottom: 5rem;
}
.wip {
	opacity: 0.2 !important;
	display: none !important;
}
</style>
