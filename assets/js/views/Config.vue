<template>
	<div class="root safe-area-inset">
		<div class="container px-4">
			<TopHeader :title="$t('config.main.title')" />
			<div class="wrapper pb-5">
				<WelcomeBanner v-if="loadpointsRequired" />
				<ExperimentalBanner v-else-if="$hiddenFeatures()" />

				<h2 class="my-4 mt-5">{{ $t("config.section.general") }}</h2>
				<GeneralConfig @site-changed="siteChanged" />

				<div v-if="$hiddenFeatures()">
					<h2 class="my-4">{{ $t("config.section.loadpoints") }} 🧪</h2>
					<p
						v-if="loadpointsRequired"
						class="text-muted my-4"
						data-testid="loadpoint-required"
					>
						{{ $t("config.main.loadpointRequired") }}
					</p>
					<div class="p-0 config-list">
						<DeviceCard
							v-for="loadpoint in loadpoints"
							:key="loadpoint.name"
							:title="loadpoint.title"
							:name="loadpoint.name"
							:editable="!!loadpoint.id"
							:error="hasDeviceError('loadpoint', loadpoint.name)"
							data-testid="loadpoint"
							@edit="editLoadpoint(loadpoint.id)"
						>
							<template #tags>
								<DeviceTags :tags="loadpointTags(loadpoint)" />
							</template>
							<template #icon>
								<VehicleIcon
									v-if="chargerIcon(loadpoint.charger)"
									:name="chargerIcon(loadpoint.charger)"
								/>
								<LoadpointIcon v-else />
							</template>
						</DeviceCard>

						<NewDeviceButton
							data-testid="add-loadpoint"
							:title="$t('config.main.addLoadpoint')"
							:attention="loadpointsRequired"
							@click="newLoadpoint"
						/>
					</div>

					<h2 class="my-4">{{ $t("config.section.vehicles") }} 🧪</h2>
					<div class="p-0 config-list">
						<DeviceCard
							v-for="vehicle in vehicles"
							:key="vehicle.name"
							:title="vehicle.config?.title || vehicle.name"
							:name="vehicle.name"
							:editable="vehicle.id >= 0"
							:error="hasDeviceError('vehicle', vehicle.name)"
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
						<NewDeviceButton
							data-testid="add-vehicle"
							:title="$t('config.main.addVehicle')"
							@click="newVehicle"
						/>
					</div>

					<h2 class="my-4 mt-5">{{ $t("config.section.grid") }} 🧪</h2>
					<div class="p-0 config-list">
						<DeviceCard
							v-if="gridMeter"
							:title="$t('config.grid.title')"
							:name="gridMeter.name"
							:editable="!!gridMeter.id"
							:error="hasDeviceError('meter', gridMeter.name)"
							data-testid="grid"
							@edit="editMeter('grid', gridMeter.id)"
						>
							<template #icon>
								<shopicon-regular-powersupply></shopicon-regular-powersupply>
							</template>
							<template #tags>
								<DeviceTags :tags="deviceTags('meter', gridMeter.name)" />
							</template>
						</DeviceCard>
						<NewDeviceButton
							v-else
							:title="$t('config.main.addGrid')"
							data-testid="add-grid"
							@click="newMeter('grid')"
						/>
						<DeviceCard
							v-if="tariffTags"
							:title="$t('config.tariffs.title')"
							editable
							:error="hasClassError('tariff')"
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
						<NewDeviceButton
							v-else
							:title="$t('config.main.addTariffs')"
							data-testid="add-tariffs"
							@click="openModal('tariffsModal')"
						/>
					</div>
					<h2 class="my-4 mt-5">{{ $t("config.section.meter") }} 🧪</h2>
					<div class="p-0 config-list">
						<DeviceCard
							v-for="meter in pvMeters"
							:key="meter.name"
							:title="
								meter.deviceTitle ||
								meter.config?.template ||
								$t('config.devices.solarSystem')
							"
							:name="meter.name"
							:editable="!!meter.id"
							:error="hasDeviceError('meter', meter.name)"
							data-testid="pv"
							@edit="editMeter('pv', meter.id)"
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
							:title="
								meter.deviceTitle ||
								meter.config?.template ||
								$t('config.devices.batteryStorage')
							"
							:name="meter.name"
							:editable="!!meter.id"
							:error="hasDeviceError('meter', meter.name)"
							data-testid="battery"
							@edit="editMeter('battery', meter.id)"
						>
							<template #icon>
								<shopicon-regular-batterythreequarters></shopicon-regular-batterythreequarters>
							</template>
							<template #tags>
								<DeviceTags :tags="deviceTags('meter', meter.name)" />
							</template>
						</DeviceCard>
						<NewDeviceButton
							:title="$t('config.main.addPvBattery')"
							@click="addSolarBatteryMeter"
						/>
					</div>

					<h2 class="my-4 mt-5">{{ $t("config.section.additionalMeter") }} 🧪</h2>
					<div class="p-0 config-list">
						<DeviceCard
							v-for="meter in auxMeters"
							:key="meter.name"
							:title="
								meter.deviceTitle ||
								meter.config?.template ||
								$t('config.devices.auxMeter')
							"
							:name="meter.name"
							:editable="!!meter.id"
							:error="hasDeviceError('meter', meter.name)"
							data-testid="aux"
							@edit="editMeter('aux', meter.id)"
						>
							<template #icon>
								<VehicleIcon :name="meter.deviceIcon || 'smartconsumer'" />
							</template>
							<template #tags>
								<DeviceTags :tags="deviceTags('meter', meter.name)" />
							</template>
						</DeviceCard>
						<NewDeviceButton
							:title="$t('config.main.addAdditional')"
							@click="newAdditionalMeter"
						/>
					</div>

					<h2 class="my-4 mt-5">{{ $t("config.section.integrations") }} 🧪</h2>

					<div class="p-0 config-list">
						<DeviceCard
							:title="$t('config.mqtt.title')"
							editable
							:error="hasClassError('mqtt')"
							data-testid="mqtt"
							@edit="openModal('mqttModal')"
						>
							<template #icon><MqttIcon /></template>
							<template #tags>
								<DeviceTags :tags="mqttTags" />
							</template>
						</DeviceCard>
						<DeviceCard
							:title="$t('config.messaging.title')"
							editable
							:error="hasClassError('messenger')"
							data-testid="messaging"
							@edit="openModal('messagingModal')"
						>
							<template #icon><NotificationIcon /></template>
							<template #tags>
								<DeviceTags :tags="messagingTags" />
							</template>
						</DeviceCard>
						<DeviceCard
							:title="$t('config.influx.title')"
							editable
							:error="hasClassError('influx')"
							data-testid="influx"
							@edit="openModal('influxModal')"
						>
							<template #icon><InfluxIcon /></template>
							<template #tags>
								<DeviceTags :tags="influxTags" />
							</template>
						</DeviceCard>
						<DeviceCard
							:title="`${$t('config.eebus.title')} 🧪`"
							editable
							:error="hasClassError('eebus')"
							data-testid="eebus"
							@edit="openModal('eebusModal')"
						>
							<template #icon><EebusIcon /></template>
							<template #tags>
								<DeviceTags :tags="eebusTags" />
							</template>
						</DeviceCard>

						<DeviceCard
							:title="`${$t('config.circuits.title')} 🧪`"
							editable
							:error="hasClassError('circuit')"
							data-testid="circuits"
							@edit="openModal('circuitsModal')"
						>
							<template #icon><CircuitsIcon /></template>
							<template #tags>
								<DeviceTags
									v-if="circuits.length == 0"
									:tags="{ configured: { value: false } }"
								/>
								<template
									v-for="(circuit, idx) in circuits"
									v-else
									:key="circuit.name"
								>
									<hr v-if="idx > 0" />
									<p class="my-2 fw-bold">
										{{ circuit.config?.title }}
										<code>({{ circuit.name }})</code>
									</p>
									<DeviceTags :tags="circuitTags(circuit)" />
								</template>
							</template>
						</DeviceCard>
						<DeviceCard
							:title="$t('config.modbusproxy.title')"
							editable
							:error="hasClassError('modbusproxy')"
							data-testid="modbusproxy"
							@edit="openModal('modbusProxyModal')"
						>
							<template #icon><ModbusProxyIcon /></template>
							<template #tags>
								<DeviceTags :tags="modbusproxyTags" />
							</template>
						</DeviceCard>
						<DeviceCard
							:title="$t('config.hems.title')"
							editable
							:error="hasClassError('hems')"
							data-testid="hems"
							@edit="openModal('hemsModal')"
						>
							<template #icon><HemsIcon /></template>
							<template #tags>
								<DeviceTags :tags="hemsTags" />
							</template>
						</DeviceCard>
					</div>
				</div>

				<hr class="my-5" />

				<h2 class="my-4 mt-5">{{ $t("config.section.system") }}</h2>
				<div class="round-box p-4 d-flex gap-4 mb-5 flex-wrap">
					<router-link to="/log" class="btn btn-outline-secondary">
						{{ $t("config.system.logs") }}
					</router-link>
					<button
						class="btn btn-outline-secondary text-truncate"
						@click="openModal('backupRestoreModal')"
					>
						{{ $t("config.system.backupRestore.title") }}
					</button>
					<button class="btn btn-outline-danger" @click="restart">
						{{ $t("config.system.restart") }}
					</button>
				</div>

				<LoadpointModal
					:id="selectedLoadpointId"
					ref="loadpointModal"
					:vehicleOptions="vehicleOptions"
					:loadpointCount="loadpoints.length"
					:chargers="chargers"
					:chargerValues="deviceValues['charger']"
					:meters="meters"
					:circuits="circuits"
					:fade="loadpointSubModalOpen ? 'left' : ''"
					:hasDeviceError="hasDeviceError"
					@updated="loadpointChanged"
					@open-charger-modal="editLoadpointCharger"
					@open-meter-modal="editLoadpointMeter"
					@opened="loadpointSubModalOpen = false"
				/>
				<VehicleModal :id="selectedVehicleId" @vehicle-changed="vehicleChanged" />
				<MeterModal
					:id="selectedMeterId"
					:name="selectedMeterName"
					:type="selectedMeterType"
					:typeChoices="selectedMeterTypeChoices"
					:fade="loadpointSubModalOpen ? 'right' : ''"
					@added="meterAdded"
					@updated="meterChanged"
					@removed="meterRemoved"
					@close="meterModalClosed"
				/>
				<ChargerModal
					:id="selectedChargerId"
					:name="selectedChargerName"
					:loadpointType="selectedLoadpointType"
					:fade="loadpointSubModalOpen ? 'right' : ''"
					:isSponsor="isSponsor"
					@added="chargerAdded"
					@updated="chargerChanged"
					@removed="chargerRemoved"
					@close="chargerModalClosed"
				/>
				<InfluxModal @changed="loadDirty" />
				<MqttModal @changed="loadDirty" />
				<NetworkModal @changed="loadDirty" />
				<ControlModal @changed="loadDirty" />
				<SponsorModal @changed="loadDirty" />
				<HemsModal @changed="yamlChanged" />
				<MessagingModal @changed="yamlChanged" />
				<TariffsModal @changed="yamlChanged" />
				<ModbusProxyModal @changed="yamlChanged" />
				<CircuitsModal @changed="yamlChanged" />
				<EebusModal @changed="yamlChanged" />
				<BackupRestoreModal v-bind="backupRestoreProps" />
				<PasswordModal update-mode />
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/receivepayment";
import NewDeviceButton from "../components/Config/NewDeviceButton.vue";
import api from "../api";
import ChargerModal from "../components/Config/ChargerModal.vue";
import CircuitsIcon from "../components/MaterialIcon/Circuits.vue";
import CircuitsModal from "../components/Config/CircuitsModal.vue";
import collector from "../mixins/collector";
import ControlModal from "../components/Config/ControlModal.vue";
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
import LoadpointModal from "../components/Config/LoadpointModal.vue";
import LoadpointIcon from "../components/MaterialIcon/Loadpoint.vue";
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
import SponsorModal from "../components/Config/SponsorModal.vue";
import store from "../store";
import TariffsModal from "../components/Config/TariffsModal.vue";
import Header from "../components/Top/Header.vue";
import VehicleIcon from "../components/VehicleIcon";
import VehicleModal from "../components/Config/VehicleModal.vue";
import { defineComponent } from "vue";
import type {
	ConfigCharger,
	ConfigVehicle,
	ConfigCircuit,
	ConfigLoadpoint,
	ConfigMeter,
	LoadpointType,
	Timeout,
	SelectedMeterType,
	SiteConfig,
	DeviceType,
} from "@/types/evcc";

type DeviceValuesMap = Record<DeviceType, Record<string, any>>;
import BackupRestoreModal from "@/components/Config/BackupRestoreModal.vue";
import WelcomeBanner from "../components/Config/WelcomeBanner.vue";
import ExperimentalBanner from "../components/Config/ExperimentalBanner.vue";
import PasswordModal from "../components/Auth/PasswordModal.vue";

export default defineComponent({
	name: "Config",
	components: {
		NewDeviceButton,
		BackupRestoreModal,
		ChargerModal,
		CircuitsIcon,
		CircuitsModal,
		ControlModal,
		DeviceCard,
		DeviceTags,
		EebusIcon,
		EebusModal,
		ExperimentalBanner,
		GeneralConfig,
		HemsIcon,
		HemsModal,
		InfluxIcon,
		InfluxModal,
		MessagingModal,
		MeterModal,
		LoadpointModal,
		LoadpointIcon,
		ModbusProxyIcon,
		ModbusProxyModal,
		MqttIcon,
		MqttModal,
		NetworkModal,
		NotificationIcon,
		SponsorModal,
		TariffsModal,
		TopHeader: Header,
		VehicleIcon,
		VehicleModal,
		WelcomeBanner,
		PasswordModal,
	},
	mixins: [formatter, collector],
	props: {
		offline: Boolean,
		notifications: Array,
	},
	data() {
		return {
			vehicles: [] as ConfigVehicle[],
			meters: [] as ConfigMeter[],
			loadpoints: [] as ConfigLoadpoint[],
			chargers: [] as ConfigCharger[],
			circuits: [] as ConfigCircuit[],
			selectedVehicleId: undefined as number | undefined,
			selectedMeterId: undefined as number | undefined,
			selectedMeterType: undefined as SelectedMeterType | undefined,
			selectedMeterTypeChoices: [] as string[],
			selectedChargerId: undefined as number | undefined,
			selectedLoadpointId: undefined as number | undefined,
			selectedLoadpointType: undefined as LoadpointType | undefined,
			loadpointSubModalOpen: false,
			site: {
				grid: "",
				pv: [] as string[],
				battery: [] as string[],
				title: "",
				aux: null as string[] | null,
				ext: null as string[] | null,
			} as SiteConfig,
			deviceValueTimeout: null as Timeout,
			deviceValues: {
				meter: {},
				vehicle: {},
				charger: {},
				loadpoint: {},
			} as DeviceValuesMap,
			isComponentMounted: true,
			isPageVisible: true,
		};
	},
	head() {
		return { title: this.$t("config.main.title") };
	},
	computed: {
		loadpointsRequired() {
			return this.loadpoints.length === 0;
		},
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
		auxMeters() {
			const names = this.site?.aux;
			return this.getMetersByNames(names);
		},
		extMeters() {
			const names = this.site?.ext;
			return this.getMetersByNames(names);
		},
		selectedMeterName() {
			return this.getMeterById(this.selectedMeterId)?.name;
		},
		selectedChargerName() {
			return this.getChargerById(this.selectedChargerId)?.name;
		},
		tariffTags() {
			const { currency, tariffGrid, tariffFeedIn, tariffCo2, tariffSolar } = store.state;
			if (
				tariffGrid === undefined &&
				tariffFeedIn === undefined &&
				tariffCo2 === undefined &&
				tariffSolar === undefined
			) {
				return null;
			}
			const tags = {
				currency: {},
				gridPrice: {},
				feedinPrice: {},
				co2: {},
				solarForecast: {},
			};
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
			if (tariffSolar) {
				tags.solarForecast = { value: tariffSolar };
			}
			return tags;
		},
		mqttTags() {
			const { broker, topic } = store.state?.mqtt || {};
			if (!broker) return { configured: { value: false } };
			return {
				broker: { value: broker },
				topic: { value: topic },
			};
		},
		influxTags() {
			const { url, database, org } = store.state?.influx || {};
			if (!url) return { configured: { value: false } };
			const result = { url: { value: url }, bucket: {}, org: {} };
			if (database) result.bucket = { value: database };
			if (org) result.org = { value: org };
			return result;
		},
		vehicleOptions() {
			return this.vehicles.map((v) => ({ key: v.name, name: v.config?.title || v.name }));
		},
		hemsTags() {
			const result = { configured: { value: false }, hemsType: {} };
			const { type } = store.state?.hems || {};
			if (type) {
				result.configured.value = true;
				result.hemsType = { value: type };
			}
			return result;
		},
		isSponsor() {
			const { name } = store.state?.sponsor || {};
			return !!name;
		},
		eebusTags() {
			return { configured: { value: store.state?.eebus || false } };
		},
		modbusproxyTags() {
			const config = store.state?.modbusproxy || [];
			if (config.length > 0) {
				return { amount: { value: config.length } };
			}
			return { configured: { value: false } };
		},
		messagingTags() {
			return { configured: { value: store.state?.messaging || false } };
		},
		backupRestoreProps() {
			return {
				authDisabled: store.state?.authDisabled || false,
			};
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
		this.isComponentMounted = true;
		document.addEventListener("visibilitychange", this.handleVisibilityChange);
		this.isPageVisible = document.visibilityState === "visible";
		this.loadAll();
	},
	unmounted() {
		this.isComponentMounted = false;
		document.removeEventListener("visibilitychange", this.handleVisibilityChange);
		if (this.deviceValueTimeout) {
			clearTimeout(this.deviceValueTimeout);
		}
	},
	methods: {
		handleVisibilityChange() {
			this.isPageVisible = document.visibilityState === "visible";
			if (this.isPageVisible) {
				this.updateValues();
			} else if (this.deviceValueTimeout) {
				clearTimeout(this.deviceValueTimeout);
			}
		},
		async loadAll() {
			await this.loadVehicles();
			await this.loadMeters();
			await this.loadSite();
			await this.loadChargers();
			await this.loadLoadpoints();
			await this.loadCircuits();
			await this.loadDirty();
			this.updateValues();
		},
		async loadDirty() {
			const response = await api.get("/config/dirty");
			if (response.data) {
				restart.restartNeeded = true;
			}
		},
		async loadVehicles() {
			const response = await api.get("/config/devices/vehicle");
			this.vehicles = response.data || [];
		},
		async loadChargers() {
			const response = await api.get("/config/devices/charger");
			this.chargers = response.data || [];
		},
		async loadMeters() {
			const response = await api.get("/config/devices/meter");
			this.meters = response.data || [];
		},
		async loadCircuits() {
			const response = await api.get("/config/devices/circuit");
			this.circuits = response.data || [];
		},
		async loadSite() {
			const response = await api.get("/config/site", {
				validateStatus: (status: number) => status < 500,
			});
			if (response.status === 200) {
				this.site = response.data;
			}
		},
		async loadLoadpoints() {
			const response = await api.get("/config/loadpoints");
			this.loadpoints = response.data || [];
		},
		getMetersByNames(names: string[] | null) {
			if (!names || !this.meters) {
				return [];
			}
			return this.meters.filter((m) => names.includes(m.name));
		},
		getMeterById(id?: number) {
			if (!id || !this.meters) {
				return undefined;
			}
			return this.meters.find((m) => m.id === id);
		},
		getChargerById(id?: number) {
			if (!id || !this.chargers) {
				return undefined;
			}
			return this.chargers.find((c) => c.id === id);
		},
		vehicleModal() {
			return Modal.getOrCreateInstance(
				document.getElementById("vehicleModal") as HTMLElement
			);
		},
		meterModal() {
			return Modal.getOrCreateInstance(document.getElementById("meterModal") as HTMLElement);
		},
		loadpointModal() {
			return Modal.getOrCreateInstance(
				document.getElementById("loadpointModal") as HTMLElement
			);
		},
		chargerModal() {
			return Modal.getOrCreateInstance(
				document.getElementById("chargerModal") as HTMLElement
			);
		},
		editLoadpointCharger(name: string, loadpointType?: LoadpointType) {
			this.loadpointSubModalOpen = true;
			const charger = this.chargers.find((c) => c.name === name);
			if (charger && charger.id === undefined) {
				alert(
					"yaml configured chargers can not be edited. Remove charger from yaml first."
				);
				return;
			}
			this.loadpointModal().hide();
			this.$nextTick(() => this.editCharger(charger?.id, loadpointType));
		},
		editLoadpointMeter(name: string) {
			this.loadpointSubModalOpen = true;
			const meter = this.meters.find((m) => m.name === name);
			if (meter && meter.id === undefined) {
				alert("yaml configured meters can not be edited. Remove meter from yaml first.");
				return;
			}
			this.loadpointModal().hide();
			this.$nextTick(() => this.editMeter("charge", meter?.id));
		},
		editMeter(type: SelectedMeterType, id?: number) {
			this.selectedMeterType = type;
			this.selectedMeterId = id;
			this.$nextTick(() => this.meterModal().show());
		},
		newMeter(type: SelectedMeterType) {
			this.selectedMeterId = undefined;
			this.selectedMeterType = type;
			this.$nextTick(() => this.meterModal().show());
		},
		addSolarBatteryMeter() {
			this.selectedMeterId = undefined;
			this.selectedMeterType = undefined;
			this.selectedMeterTypeChoices = ["pv", "battery"];
			this.$nextTick(() => this.meterModal().show());
		},
		newAdditionalMeter() {
			this.selectedMeterId = undefined;
			this.selectedMeterType = undefined;
			this.selectedMeterTypeChoices = ["aux", "ext"];
			this.$nextTick(() => this.meterModal().show());
		},
		editCharger(id?: number, loadpointType?: LoadpointType) {
			this.selectedChargerId = id;
			this.selectedLoadpointType = loadpointType;
			this.$nextTick(() => this.chargerModal().show());
		},
		async meterChanged() {
			await this.loadMeters();
			await this.loadDirty();
			this.updateValues();
		},
		async chargerChanged() {
			await this.loadChargers();
			await this.loadDirty();
			this.updateValues();
		},
		editLoadpoint(id?: number) {
			if (!id) alert("missing loadpoint id");
			this.selectedLoadpointId = id;
			this.$nextTick(() => this.loadpointModal().show());
		},
		newLoadpoint() {
			this.selectedLoadpointId = undefined;
			(
				this.$refs["loadpointModal"] as InstanceType<typeof LoadpointModal> | undefined
			)?.reset();
			this.$nextTick(() => this.loadpointModal().show());
		},
		async loadpointChanged() {
			this.selectedLoadpointId = undefined;
			await this.loadLoadpoints();
			this.loadDirty();
		},
		editVehicle(id: number) {
			this.selectedVehicleId = id;
			this.$nextTick(() => this.vehicleModal().show());
		},
		newVehicle() {
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
		yamlChanged() {
			this.loadDirty();
		},
		meterAdded(type: SelectedMeterType, name: string) {
			if (type === "charge") {
				// update loadpoint
				(
					this.$refs["loadpointModal"] as InstanceType<typeof LoadpointModal> | undefined
				)?.setMeter(name);
			} else if (type === "grid") {
				// update site grid
				this.site.grid = name;
				this.saveSite(type);
			} else {
				// update site pv, battery, aux, ext with type-safe approach
				switch (type) {
					case "pv":
						if (!this.site.pv) this.site.pv = [];
						this.site.pv.push(name);
						break;
					case "battery":
						if (!this.site.battery) this.site.battery = [];
						this.site.battery.push(name);
						break;
					case "aux":
						if (!this.site.aux) this.site.aux = [];
						this.site.aux.push(name);
						break;
					case "ext":
						if (!this.site.ext) this.site.ext = [];
						this.site.ext.push(name);
						break;
				}
				this.saveSite(type);
			}
			this.meterChanged();
		},
		meterRemoved(type: SelectedMeterType) {
			if (type === "charge") {
				// update loadpoint
				(
					this.$refs["loadpointModal"] as InstanceType<typeof LoadpointModal> | undefined
				)?.setMeter("");
			} else {
				// update site grid, pv, battery, aux, ext
				this.loadSite();
				this.loadDirty();
			}
			this.meterChanged();
		},
		async chargerAdded(name: string) {
			await this.chargerChanged();
			(
				this.$refs["loadpointModal"] as InstanceType<typeof LoadpointModal> | undefined
			)?.setCharger(name);
		},
		chargerRemoved() {
			(
				this.$refs["loadpointModal"] as InstanceType<typeof LoadpointModal> | undefined
			)?.setCharger("");
			this.chargerChanged();
		},
		meterModalClosed() {
			if (this.selectedMeterType === "charge") {
				// reopen loadpoint modal
				this.loadpointModal().show();
			}
		},
		chargerModalClosed() {
			// reopen loadpoint modal
			this.loadpointModal().show();
		},
		async saveSite(key: keyof SiteConfig) {
			const body = key ? { [key]: this.site[key] } : this.site;
			await api.put("/config/site", body);
			await this.loadSite();
			await this.loadDirty();
			this.updateValues();
		},
		todo() {
			alert("not implemented yet");
		},
		async restart() {
			await performRestart();
		},
		async updateDeviceValue(type: DeviceType, name: string) {
			try {
				const response = await api.get(`/config/devices/${type}/${name}/status`);
				if (!this.deviceValues[type]) this.deviceValues[type] = {};
				this.deviceValues[type][name] = response.data;
			} catch (error) {
				console.error("Error fetching device values for", type, name, error);
			}
		},
		async updateValues() {
			if (this.deviceValueTimeout) {
				clearTimeout(this.deviceValueTimeout);
			}
			if (!this.offline) {
				const devices = {
					meter: this.meters,
					vehicle: this.vehicles,
					charger: this.chargers,
				} as Record<DeviceType, any[]>;
				for (const type in devices) {
					for (const device of devices[type as DeviceType]) {
						if (this.isComponentMounted && this.isPageVisible) {
							await this.updateDeviceValue(type as DeviceType, device.name);
						}
					}
				}
			}

			if (this.isComponentMounted && this.isPageVisible) {
				const interval = (store.state?.interval || 30) * 1000;
				this.deviceValueTimeout = setTimeout(this.updateValues, interval);
			}
		},
		deviceTags(type: DeviceType, id: string) {
			return this.deviceValues[type][id] || {};
		},
		loadpointTags(loadpoint: ConfigLoadpoint) {
			const { charger, meter } = loadpoint;
			const chargerTags = charger ? this.deviceTags("charger", charger) : {};
			const meterTags = meter ? this.deviceTags("meter", meter) : {};
			return { ...chargerTags, ...meterTags };
		},
		openModal(id: string) {
			const $el = document.getElementById(id);
			if ($el) {
				Modal.getOrCreateInstance($el).show();
			} else {
				console.error(`modal ${id} not found`);
			}
		},
		circuitTags(circuit: ConfigCircuit) {
			const circuits = store.state?.circuits || {};
			const data = circuits[circuit.name] || {};
			const result: Record<string, object> = {};
			const p = data.power || 0;
			if (data.maxPower) {
				result["powerRange"] = {
					value: [p, data.maxPower],
					warning: data.power && data.power >= data.maxPower,
				};
			} else {
				result["power"] = { value: p, muted: true };
			}
			if (data.maxCurrent) {
				result["currentRange"] = {
					value: [data.current || 0, data.maxCurrent],
					warning: data.current && data.current >= data.maxCurrent,
				};
			}
			return result;
		},
		hasDeviceError(type: DeviceType, name?: string) {
			if (!name) return false;
			const fatals = store.state?.fatal || [];
			return fatals.some((fatal) => fatal.class === type && fatal.device === name);
		},
		hasClassError(className: string) {
			const fatals = store.state?.fatal || [];
			return fatals.some((fatal) => fatal.class === className);
		},
		chargerIcon(chargerName: string) {
			const charger = this.chargers.find((c) => c.name === chargerName);

			return charger?.config?.icon || this.deviceValues["charger"][chargerName]?.icon?.value;
		},
	},
});
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
