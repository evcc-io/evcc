<template>
	<div class="root safe-area-inset">
		<div class="container px-4">
			<TopHeader
				ref="header"
				:title="$t('config.main.title')"
				:notifications="notifications"
			/>
			<div class="wrapper pb-5">
				<AuthSuccessBanner
					v-if="callbackCompleted || callbackError"
					:provider-id="callbackCompleted"
					:error="callbackError"
					:auth-providers="authProviders"
				/>

				<h2 class="my-4 mt-5">{{ $t("config.section.general") }}</h2>
				<GeneralConfig
					:experimental="experimental"
					:sponsor-error="hasClassError('sponsorship')"
					@site-changed="siteChanged"
				/>

				<WelcomeBanner v-if="setupRequired" />
				<h2 class="my-4">{{ $t("config.section.loadpoints") }}</h2>
				<div class="p-0 config-list">
					<DeviceCard
						v-for="loadpoint in loadpoints"
						:key="loadpoint.name"
						:title="loadpoint.title"
						:name="loadpoint.name"
						:editable="!!loadpoint.id"
						:error="hasDeviceError('loadpoint', loadpoint.name)"
						data-testid="loadpoint"
						@edit="openModal('loadpoint', { id: loadpoint.id })"
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
						@click="openModal('loadpoint')"
					/>
				</div>

				<h2 class="my-4">{{ $t("config.section.vehicles") }}</h2>
				<div class="p-0 config-list">
					<DeviceCard
						v-for="vehicle in vehicles"
						:key="vehicle.name"
						:title="vehicle.config?.title || vehicle.name"
						:name="vehicle.name"
						:editable="vehicle.id >= 0"
						:error="hasDeviceError('vehicle', vehicle.name)"
						data-testid="vehicle"
						@edit="openModal('vehicle', { id: vehicle.id })"
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
						@click="openModal('vehicle')"
					/>
				</div>

				<h2 class="my-4 mt-5">{{ $t("config.section.grid") }}</h2>
				<div class="p-0 config-list">
					<MeterCard
						v-if="gridMeter"
						:meter="gridMeter"
						:title="$t('config.grid.title')"
						meter-type="grid"
						:has-error="hasDeviceError('meter', gridMeter.name)"
						:tags="deviceTags('meter', gridMeter.name)"
						@edit="(type, id) => openModal('meter', { type, id })"
					/>
					<NewDeviceButton
						v-else
						:title="$t('config.main.addGrid')"
						data-testid="add-grid"
						@click="openModal('meter', { type: 'grid' })"
					/>
					<DeviceCard
						:title="$t('config.tariffs.title')"
						editable
						:unconfigured="isUnconfigured(tariffTags)"
						:error="hasClassError('tariff')"
						data-testid="tariffs"
						@edit="openModal('tariffs')"
					>
						<template #icon>
							<shopicon-regular-receivepayment></shopicon-regular-receivepayment>
						</template>
						<template #tags>
							<DeviceTags :tags="tariffTags" />
						</template>
					</DeviceCard>
				</div>
				<h2 class="my-4 mt-5">{{ $t("config.section.meter") }}</h2>
				<div class="p-0 config-list">
					<MeterCard
						v-for="meter in pvMeters"
						:key="meter.name"
						:meter="meter"
						meter-type="pv"
						:has-error="hasDeviceError('meter', meter.name)"
						:tags="deviceTags('meter', meter.name)"
						@edit="(type, id) => openModal('meter', { type, id })"
					/>
					<MeterCard
						v-for="meter in batteryMeters"
						:key="meter.name"
						:meter="meter"
						meter-type="battery"
						:has-error="hasDeviceError('meter', meter.name)"
						:tags="deviceTags('meter', meter.name)"
						@edit="(type, id) => openModal('meter', { type, id })"
					/>
					<NewDeviceButton
						:title="$t('config.main.addPvBattery')"
						@click="openModal('meter', { choices: ['pv', 'battery'] })"
					/>
				</div>

				<h2 class="my-4 mt-5">{{ $t("config.section.additionalMeter") }}</h2>
				<div class="p-0 config-list">
					<MeterCard
						v-for="meter in auxMeters"
						:key="meter.name"
						:meter="meter"
						meter-type="aux"
						:has-error="hasDeviceError('meter', meter.name)"
						:tags="deviceTags('meter', meter.name)"
						@edit="(type, id) => openModal('meter', { type, id })"
					/>
					<MeterCard
						v-for="meter in extMeters"
						:key="meter.name"
						:meter="meter"
						meter-type="ext"
						:has-error="hasDeviceError('meter', meter.name)"
						:tags="deviceTags('meter', meter.name)"
						@edit="(type, id) => openModal('meter', { type, id })"
					/>
					<NewDeviceButton
						:title="$t('config.main.addAdditional')"
						@click="openModal('meter', { choices: ['aux', 'ext'] })"
					/>
				</div>

				<h2 class="my-4 mt-5">{{ $t("config.section.integrations") }}</h2>
				<div class="p-0 config-list">
					<AuthProvidersCard
						:providers="authProviders"
						data-testid="auth-providers"
						@auth-request="handleProviderAuthRequest"
					/>
					<DeviceCard
						:title="$t('config.mqtt.title')"
						editable
						:error="hasClassError('mqtt')"
						:unconfigured="isUnconfigured(mqttTags)"
						data-testid="mqtt"
						@edit="openModal('mqtt')"
					>
						<template #icon><MqttIcon /></template>
						<template #tags>
							<DeviceTags :tags="mqttTags" />
						</template>
					</DeviceCard>
					<DeviceCard
						:title="$t('config.messaging.title')"
						:editable="messagingYamlSource !== 'file'"
						:error="hasClassError('messenger')"
						:unconfigured="isUnconfigured(messagingTags)"
						:badge="messagingYamlSource === 'db'"
						data-testid="messaging"
						@edit="openMessagingModal"
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
						:unconfigured="isUnconfigured(influxTags)"
						data-testid="influx"
						@edit="openModal('influx')"
					>
						<template #icon><InfluxIcon /></template>
						<template #tags>
							<DeviceTags :tags="influxTags" />
						</template>
					</DeviceCard>
					<DeviceCard
						:title="`${$t('config.circuits.title')}`"
						editable
						:error="hasClassError('circuit')"
						:unconfigured="circuitsSorted.length === 0"
						data-testid="circuits"
						@edit="openModal('circuits')"
					>
						<template #icon><CircuitsIcon /></template>
						<template #tags>
							<DeviceTags
								v-if="circuitsSorted.length == 0"
								:tags="{ configured: { value: false } }"
							/>
							<template
								v-for="(circuit, idx) in circuitsSorted"
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
						:unconfigured="isUnconfigured(modbusproxyTags)"
						data-testid="modbusproxy"
						@edit="openModal('modbusproxy')"
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
						:unconfigured="isUnconfigured(hemsTags)"
						data-testid="hems"
						@edit="openModal('hems')"
					>
						<template #icon><HemsIcon /></template>
						<template #tags>
							<DeviceTags :tags="hemsTags" />
						</template>
					</DeviceCard>
				</div>

				<h2 class="my-4 mt-5">{{ $t("config.section.services") }}</h2>
				<div class="p-0 config-list">
					<DeviceCard
						:title="$t('config.ocpp.title')"
						editable
						:error="hasClassError('ocpp')"
						data-testid="ocpp"
						@edit="openModal('ocpp')"
					>
						<template #icon><OcppIcon /></template>
					</DeviceCard>
					<DeviceCard
						:title="$t('config.shm.cardTitle')"
						editable
						:error="hasClassError('shm')"
						data-testid="shm"
						@edit="openModal('shm')"
					>
						<template #icon><ShmIcon /></template>
					</DeviceCard>
					<DeviceCard
						:title="$t('config.eebus.title')"
						editable
						:error="hasClassError('eebus')"
						data-testid="eebus"
						@edit="openModal('eebus')"
					>
						<template #icon><EebusIcon /></template>
					</DeviceCard>
				</div>

				<hr class="my-5" />

				<h2 class="my-4 mt-5">{{ $t("config.section.system") }}</h2>
				<div class="round-box p-4 d-flex gap-4 mb-5 flex-wrap">
					<router-link to="/log" class="btn btn-outline-secondary">
						{{ $t("config.system.logs") }}
					</router-link>
					<router-link to="/issue" class="btn btn-outline-secondary">
						{{ $t("help.issueButton") }}
					</router-link>
					<button
						data-testid="backup-restore"
						class="btn btn-outline-secondary text-truncate"
						@click="openModal('backuprestore')"
					>
						{{ $t("config.system.backupRestore.title") }}
					</button>
					<button class="btn btn-outline-danger" @click="restart">
						{{ $t("config.system.restart") }}
					</button>
				</div>

				<LoadpointModal
					:vehicleOptions="vehicleOptions"
					:loadpointCount="loadpoints.length"
					:chargers="chargers"
					:chargerValues="deviceValues['charger']"
					:meters="meters"
					:circuits="circuits"
					:hasDeviceError="hasDeviceError"
					@changed="loadpointChanged"
				/>
				<VehicleModal :is-sponsor="isSponsor" @vehicle-changed="vehicleChanged" />
				<MeterModal :is-sponsor="isSponsor" @changed="meterChanged" />
				<ChargerModal :is-sponsor="isSponsor" :ocpp="ocpp" @changed="chargerChanged" />
				<InfluxModal @changed="loadDirty" />
				<MqttModal @changed="loadDirty" />
				<NetworkModal @changed="loadDirty" />
				<ControlModal @changed="loadDirty" />
				<HemsModal :yamlSource="hems?.yamlSource" @changed="loadDirty" />
				<ShmModal @changed="loadDirty" />
				<MessagingLegacyModal @changed="loadDirty" />
				<MessagingModal :messengers="messengers" @changed="loadDirty" />
				<MessengerModal @changed="messengerChanged" />
				<TariffsModal @changed="loadDirty" />
				<TelemetryModal :sponsor="sponsor" :telemetry="telemetry" />
				<ExperimentalModal :experimental="experimental" />
				<TitleModal @changed="loadDirty" />
				<ModbusProxyModal :is-sponsor="isSponsor" @changed="loadDirty" />
				<CircuitsModal :gridMeter="gridMeter" :extMeters="extMeters" @changed="loadDirty" />
				<EebusModal
					:status="eebus?.status"
					:yamlSource="eebus?.yamlSource"
					@changed="loadDirty"
				/>
				<OcppModal :ocpp="ocpp" />
				<BackupRestoreModal v-bind="backupRestoreProps" />
				<PasswordModal update-mode />
				<SponsorModal :error="hasClassError('sponsorship')" @changed="loadDirty" />
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
import OcppIcon from "../components/MaterialIcon/Ocpp.vue";
import OcppModal from "../components/Config/OcppModal.vue";
import formatter from "../mixins/formatter";
import GeneralConfig from "../components/Config/GeneralConfig.vue";
import HemsIcon from "../components/MaterialIcon/Hems.vue";
import HemsModal from "../components/Config/HemsModal.vue";
import ShmIcon from "../components/MaterialIcon/Shm.vue";
import ShmModal from "@/components/Config/ShmModal.vue";
import InfluxIcon from "../components/MaterialIcon/Influx.vue";
import InfluxModal from "../components/Config/InfluxModal.vue";
import LoadpointModal from "../components/Config/LoadpointModal.vue";
import LoadpointIcon from "../components/MaterialIcon/Loadpoint.vue";
import MessagingModal from "../components/Config/Messaging/MessagingModal.vue";
import MeterModal from "../components/Config/MeterModal.vue";
import MeterCard from "../components/Config/MeterCard.vue";
import { openModal, type ModalResult } from "@/configModal";
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
import TelemetryModal from "../components/Config/TelemetryModal.vue";
import ExperimentalModal from "../components/Config/ExperimentalModal.vue";
import TitleModal from "../components/Config/TitleModal.vue";
import Header from "../components/Top/Header.vue";
import VehicleIcon from "../components/VehicleIcon";
import VehicleModal from "../components/Config/VehicleModal.vue";
import { defineComponent, type PropType } from "vue";
import type {
	Circuit,
	ConfigCharger,
	ConfigVehicle,
	ConfigCircuit,
	ConfigLoadpoint,
	ConfigMeter,
	Timeout,
	VehicleOption,
	MeterType,
	SiteConfig,
	DeviceType,
	Notification,
	ConfigMessenger,
} from "@/types/evcc";

type DeviceValuesMap = Record<DeviceType, Record<string, any>>;

type DeviceTags = Record<
	string,
	{ value?: any; error?: boolean; warning?: boolean; muted?: boolean; options?: any }
>;

import BackupRestoreModal from "@/components/Config/BackupRestoreModal.vue";
import WelcomeBanner from "../components/Config/WelcomeBanner.vue";
import AuthSuccessBanner from "../components/Config/AuthSuccessBanner.vue";
import PasswordModal from "../components/Auth/PasswordModal.vue";
import AuthProvidersCard from "../components/Config/AuthProvidersCard.vue";
import MessengerModal from "@/components/Config/Messaging/MessengerModal.vue";
import MessagingLegacyModal from "@/components/Config/Messaging/MessagingLegacyModal.vue";

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
		OcppIcon,
		OcppModal,
		GeneralConfig,
		HemsIcon,
		HemsModal,
		ShmModal,
		ShmIcon,
		InfluxIcon,
		InfluxModal,
		MessagingLegacyModal,
		MessagingModal,
		MessengerModal,
		MeterModal,
		MeterCard,
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
		TelemetryModal,
		ExperimentalModal,
		TitleModal,
		TopHeader: Header,
		VehicleIcon,
		VehicleModal,
		WelcomeBanner,
		AuthSuccessBanner,
		PasswordModal,
		AuthProvidersCard,
	},
	mixins: [formatter, collector],
	props: {
		offline: Boolean,
		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
	},
	data() {
		return {
			messengers: [] as ConfigMessenger[],
			vehicles: [] as ConfigVehicle[],
			meters: [] as ConfigMeter[],
			loadpoints: [] as ConfigLoadpoint[],
			chargers: [] as ConfigCharger[],
			circuits: [] as ConfigCircuit[],
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
		messagingConfigured() {
			return store.state.messaging;
		},
		callbackCompleted() {
			return this.$route.query["callbackCompleted"] as string | undefined;
		},
		callbackError() {
			return this.$route.query["callbackError"] as string | undefined;
		},
		authProviders() {
			return store.state?.authProviders;
		},
		setupRequired() {
			return store.state?.setupRequired;
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
		tariffTags(): DeviceTags {
			const { currency, tariffGrid, tariffFeedIn, tariffCo2, tariffSolar } = store.state;
			if (
				tariffGrid === undefined &&
				tariffFeedIn === undefined &&
				tariffCo2 === undefined &&
				tariffSolar === undefined
			) {
				return { configured: { value: false } };
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
		mqttTags(): DeviceTags {
			const { broker, topic } = store.state?.mqtt || {};
			if (!broker) return { configured: { value: false } };
			return {
				broker: { value: broker },
				topic: { value: topic },
			};
		},
		influxTags(): DeviceTags {
			const { url, database, org } = store.state?.influx || {};
			if (!url) return { configured: { value: false } };
			const result = { url: { value: url }, bucket: {}, org: {} };
			if (database) result.bucket = { value: database };
			if (org) result.org = { value: org };
			return result;
		},
		vehicleOptions(): VehicleOption[] {
			return this.vehicles.map((v) => ({ key: v.name, name: v.config?.title || v.name }));
		},
		hems() {
			return store.state?.hems;
		},
		hemsTags(): DeviceTags {
			const type = this.hems?.config?.type;
			if (!type) {
				return { configured: { value: false } };
			}
			const result = {
				hemsType: {},
				hemsActiveLimit: { value: null as number | null },
			};
			if (["relay", "eebus"].includes(type)) {
				result.hemsType = { value: type };
			}
			const lpc = store.state?.circuits?.["lpc"];
			if (lpc) {
				const value = lpc.maxPower || null;
				result.hemsActiveLimit = { value };
			}

			return result;
		},
		sponsor() {
			return store.state?.sponsor;
		},
		isSponsor(): boolean {
			return !!this.sponsor?.status?.name;
		},
		ocpp() {
			return store.state?.ocpp;
		},
		telemetry() {
			return store.state?.telemetry;
		},
		experimental() {
			return store.state?.experimental;
		},
		eebus() {
			return store.state?.eebus;
		},
		modbusproxyTags(): DeviceTags {
			const config = store.state?.modbusproxy || [];
			if (config.length > 0) {
				return { amount: { value: config.length } };
			}
			return { configured: { value: false } };
		},
		messagingTags(): DeviceTags {
			if (this.messagingUiConfigured) {
				const events = store.state?.messagingEvents || [];
				const enabledEvents = Object.values(events).filter((e) => !e.disabled).length;

				return {
					events: { value: enabledEvents },
					messengers: { value: this.messengers.length },
				};
			}

			return { configured: { value: this.messagingYamlConfigured } };
		},
		backupRestoreProps() {
			return {
				authDisabled: store.state?.authDisabled || false,
			};
		},
		circuitsSorted() {
			const sortedNames = Object.keys(store.state?.circuits || {});
			return [...this.circuits].sort(
				(a, b) => sortedNames.indexOf(a.name) - sortedNames.indexOf(b.name)
			);
		},
		messagingYamlSource() {
			return store.state.messaging?.yamlSource;
		},
		messagingYamlConfigured() {
			return this.messagingYamlSource === "file" || this.messagingYamlSource === "db";
		},
		messagingUiConfigured() {
			return (
				this.messengers.length > 0 ||
				Object.keys(store.state.messagingEvents ?? {}).length > 0
			);
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
		isUnconfigured(tags: DeviceTags): boolean {
			return tags["configured"]?.value === false;
		},
		handleVisibilityChange() {
			this.isPageVisible = document.visibilityState === "visible";
			if (this.isPageVisible) {
				this.updateValues();
			} else if (this.deviceValueTimeout) {
				clearTimeout(this.deviceValueTimeout);
			}
		},
		async loadAll() {
			await this.loadMessengers();
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
			const data = await this.loadConfig("dirty");
			if (data) {
				restart.restartNeeded = true;
			}
		},
		async loadConfig(path: string) {
			const validateStatus = (code: number) => [200, 404].includes(code);
			const response = await api.get(`/config/${path}`, { validateStatus });
			return response.status === 200 ? response.data : undefined;
		},
		async loadMessengers() {
			this.messengers = (await this.loadConfig("devices/messenger")) || [];
		},
		async loadVehicles() {
			this.vehicles = (await this.loadConfig("devices/vehicle")) || [];
		},
		async loadChargers() {
			this.chargers = (await this.loadConfig("devices/charger")) || [];
		},
		async loadMeters() {
			this.meters = (await this.loadConfig("devices/meter")) || [];
		},
		async loadCircuits() {
			const circuits = (await this.loadConfig("devices/circuit")) || [];
			// set lpc default title
			circuits.forEach((c: ConfigCircuit) => {
				if (c.name === "lpc" && !c.config?.title) {
					c.config = c.config || {};
					c.config.title = this.$t("config.hems.title");
				}
			});
			this.circuits = circuits;
		},
		async loadSite() {
			const data = await this.loadConfig("site");
			if (data) {
				this.site = data;
			}
		},
		async loadLoadpoints() {
			this.loadpoints = (await this.loadConfig("loadpoints")) || [];
		},
		getMetersByNames(names: string[] | null): ConfigMeter[] {
			if (!names || !this.meters) {
				return [];
			}
			return names
				.map((name) => this.meters.find((m) => m.name === name))
				.filter((m): m is ConfigMeter => m !== undefined);
		},
		async meterChanged(result: ModalResult) {
			// Added: update site config
			if (result.action === "added") {
				const type = result.type as MeterType;
				const name = result.name!;

				switch (type) {
					case "grid":
						this.site.grid = name;
						this.saveSite(type);
						break;
					case "pv":
						if (!this.site.pv) this.site.pv = [];
						this.site.pv.push(name);
						this.saveSite(type);
						break;
					case "battery":
						if (!this.site.battery) this.site.battery = [];
						this.site.battery.push(name);
						this.saveSite(type);
						break;
					case "aux":
						if (!this.site.aux) this.site.aux = [];
						this.site.aux.push(name);
						this.saveSite(type);
						break;
					case "ext":
						if (!this.site.ext) this.site.ext = [];
						this.site.ext.push(name);
						this.saveSite(type);
						break;
				}
			}

			// Removed: reload site config
			if (result.action === "removed") {
				await this.loadSite();
			}
			await this.loadMeters();
			await this.loadDirty();
			this.updateValues();
		},
		async chargerChanged() {
			await this.loadChargers();
			await this.loadDirty();
			this.updateValues();
		},
		async loadpointChanged() {
			await this.loadLoadpoints();
			this.loadDirty();
		},
		vehicleChanged() {
			this.loadVehicles();
			this.loadDirty();
		},
		openMessagingModal() {
			const modalName = this.messagingYamlSource === "db" ? "messaginglegacy" : "messaging";
			openModal(modalName);
		},
		async messengerChanged() {
			this.loadMessengers();
			this.loadDirty();
		},
		siteChanged() {
			this.loadDirty();
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
				const validateStatus = (status: number) => [200, 404].includes(status);
				const response = await api.get(`/config/devices/${type}/${name}/status`, {
					validateStatus,
				});
				if (response.status === 200) {
					if (!this.deviceValues[type]) this.deviceValues[type] = {};
					this.deviceValues[type][name] = response.data;
				}
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
					messenger: this.messengers,
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
		openModal,
		circuitTags(circuit: ConfigCircuit) {
			const circuits = store.state?.circuits || {};
			const data =
				(circuits[circuit.name] as Circuit | undefined) || ({} as Partial<Circuit>);
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
		handleProviderAuthRequest(providerId: string) {
			const header = this.$refs["header"] as InstanceType<typeof Header> | undefined;
			header?.requestAuthProvider(providerId);
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
