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
						@click="newLoadpoint"
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

				<h2 class="my-4 mt-5">{{ $t("config.section.grid") }}</h2>
				<div class="p-0 config-list">
					<MeterCard
						v-if="gridMeter"
						:meter="gridMeter"
						:title="$t('config.grid.title')"
						meter-type="grid"
						:has-error="hasDeviceError('meter', gridMeter.name)"
						:tags="deviceTags('meter', gridMeter.name)"
						@edit="editMeter"
					/>
					<NewDeviceButton
						v-else
						:title="$t('config.main.addGrid')"
						data-testid="add-grid"
						@click="newMeter('grid')"
					/>
					<DeviceCard
						:title="$t('config.tariff.title')"
						editable
						:unconfigured="isUnconfigured(tariffTags)"
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
						@edit="editMeter"
					/>
					<MeterCard
						v-for="meter in batteryMeters"
						:key="meter.name"
						:meter="meter"
						meter-type="battery"
						:has-error="hasDeviceError('meter', meter.name)"
						:tags="deviceTags('meter', meter.name)"
						@edit="editMeter"
					/>
					<NewDeviceButton
						:title="$t('config.main.addPvBattery')"
						@click="addSolarBatteryMeter"
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
						@edit="editMeter"
					/>
					<MeterCard
						v-for="meter in extMeters"
						:key="meter.name"
						:meter="meter"
						meter-type="ext"
						:has-error="hasDeviceError('meter', meter.name)"
						:tags="deviceTags('meter', meter.name)"
						@edit="editMeter"
					/>
					<NewDeviceButton
						:title="$t('config.main.addAdditional')"
						@click="newAdditionalMeter"
					/>
				</div>

				<h2 class="my-4 mt-5">{{ $t("config.tariff.title") }}</h2>
				<div class="p-0 config-list">
					<TariffCard
						v-if="gridTariff"
						:tariff="gridTariff"
						tariff-type="grid"
						:has-error="hasDeviceError('tariff', gridTariff.name)"
						:tags="deviceTags('tariff', gridTariff.name)"
						@edit="editTariff('grid', gridTariff.id)"
					/>
					<TariffCard
						v-if="feedinTariff"
						:tariff="feedinTariff"
						tariff-type="feedin"
						:has-error="hasDeviceError('tariff', feedinTariff.name)"
						:tags="deviceTags('tariff', feedinTariff.name)"
						@edit="editTariff('feedin', feedinTariff.id)"
					/>
					<TariffCard
						v-if="co2Tariff"
						:tariff="co2Tariff"
						tariff-type="co2"
						:has-error="hasDeviceError('tariff', co2Tariff.name)"
						:tags="deviceTags('tariff', co2Tariff.name)"
						@edit="editTariff('co2', co2Tariff.id)"
					/>
					<TariffCard
						v-if="plannerTariff"
						:tariff="plannerTariff"
						tariff-type="planner"
						:has-error="hasDeviceError('tariff', plannerTariff.name)"
						:tags="deviceTags('tariff', plannerTariff.name)"
						@edit="editTariff('planner', plannerTariff.id)"
					/>
					<TariffCard
						v-for="tariff in solarTariffs"
						:key="tariff.name"
						:tariff="tariff"
						tariff-type="solar"
						:has-error="hasDeviceError('tariff', tariff.name)"
						:tags="deviceTags('tariff', tariff.name)"
						@edit="editTariff('solar', tariff.id)"
					/>
					<NewDeviceButton
						v-if="!gridTariff || !feedinTariff"
						:title="$t('config.tariff.addTariff')"
						@click="newTariff(null, ['grid', 'feedin'])"
					/>
					<NewDeviceButton
						v-if="!co2Tariff || !plannerTariff || solarTariffs.length === 0"
						:title="$t('config.tariff.addForecast')"
						@click="newTariff(null, ['co2', 'solar', 'planner'])"
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
						:unconfigured="isUnconfigured(messagingTags)"
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
						:unconfigured="isUnconfigured(influxTags)"
						data-testid="influx"
						@edit="openModal('influxModal')"
					>
						<template #icon><InfluxIcon /></template>
						<template #tags>
							<DeviceTags :tags="influxTags" />
						</template>
					</DeviceCard>
					<DeviceCard
						:title="$t('config.eebus.title')"
						editable
						:error="hasClassError('eebus')"
						:unconfigured="isUnconfigured(eebusTags)"
						data-testid="eebus"
						@edit="openModal('eebusModal')"
					>
						<template #icon><EebusIcon /></template>
						<template #tags>
							<DeviceTags :tags="eebusTags" />
						</template>
					</DeviceCard>
					<DeviceCard
						:title="$t('config.ocpp.title')"
						editable
						:error="hasClassError('ocpp')"
						:unconfigured="isUnconfigured(ocppTags)"
						data-testid="ocpp"
						@edit="openModal('ocppModal')"
					>
						<template #icon><OcppIcon /></template>
						<template #tags>
							<DeviceTags :tags="ocppTags" />
						</template>
					</DeviceCard>

					<DeviceCard
						:title="`${$t('config.circuits.title')}`"
						editable
						:error="hasClassError('circuit')"
						:unconfigured="circuitsSorted.length === 0"
						data-testid="circuits"
						@edit="openModal('circuitsModal')"
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
						@edit="openModal('modbusProxyModal')"
					>
						<template #icon><ModbusProxyIcon /></template>
						<template #tags>
							<DeviceTags :tags="modbusproxyTags" />
						</template>
					</DeviceCard>
					<DeviceCard
						:title="$t('config.shm.cardTitle')"
						editable
						:error="hasClassError('shm')"
						:unconfigured="isUnconfigured(shmTags)"
						data-testid="shm"
						@edit="openModal('shmModal')"
					>
						<template #icon><ShmIcon /></template>
						<template #tags>
							<DeviceTags :tags="shmTags" />
						</template>
					</DeviceCard>
					<DeviceCard
						:title="$t('config.hems.title')"
						editable
						:error="hasClassError('hems')"
						:unconfigured="isUnconfigured(hemsTags)"
						data-testid="hems"
						@edit="openModal('hemsModal')"
					>
						<template #icon><HemsIcon /></template>
						<template #tags>
							<DeviceTags :tags="hemsTags" />
						</template>
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
					:fade="loadpointSubModalOpen ? 'left' : undefined"
					:hasDeviceError="hasDeviceError"
					@updated="loadpointChanged"
					@open-charger-modal="editLoadpointCharger"
					@open-meter-modal="editLoadpointMeter"
					@opened="loadpointSubModalOpen = false"
				/>
				<VehicleModal
					:id="selectedVehicleId"
					:is-sponsor="isSponsor"
					@vehicle-changed="vehicleChanged"
				/>
				<MeterModal
					:id="selectedMeterId"
					:type="selectedMeterType"
					:typeChoices="selectedMeterTypeChoices"
					:fade="loadpointSubModalOpen ? 'right' : undefined"
					:is-sponsor="isSponsor"
					@added="meterAdded"
					@updated="meterChanged"
					@removed="meterRemoved"
					@close="meterModalClosed"
				/>
				<ChargerModal
					:id="selectedChargerId"
					:loadpointType="selectedLoadpointType"
					:fade="loadpointSubModalOpen ? 'right' : undefined"
					:is-sponsor="isSponsor"
					:ocpp="ocpp"
					@added="chargerAdded"
					@updated="chargerChanged"
					@removed="chargerRemoved"
					@close="chargerModalClosed"
				/>
				<InfluxModal @changed="loadDirty" />
				<MqttModal @changed="loadDirty" />
				<NetworkModal @changed="loadDirty" />
				<ControlModal @changed="loadDirty" />
				<HemsModal :fromYaml="hems?.fromYaml" @changed="yamlChanged" />
				<ShmModal @changed="loadDirty" />
				<MessagingModal @changed="yamlChanged" />
				<TariffsModal @changed="yamlChanged" />
				<TariffModal
					:id="selectedTariffId"
					:type="selectedTariffType"
					:type-choices="selectedTariffTypeChoices"
					@added="tariffAdded"
					@updated="tariffChanged"
					@removed="tariffRemoved"
				/>
				<TelemetryModal :sponsor="sponsor" :telemetry="telemetry" />
				<ExperimentalModal />
				<ModbusProxyModal :is-sponsor="isSponsor" @changed="loadDirty" />
				<CircuitsModal
					:gridMeter="gridMeter"
					:extMeters="extMeters"
					@changed="yamlChanged"
				/>
				<EebusModal @changed="yamlChanged" />
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
import MessagingModal from "../components/Config/MessagingModal.vue";
import MeterModal from "../components/Config/MeterModal.vue";
import MeterCard from "../components/Config/MeterCard.vue";
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
import TariffCard from "../components/Config/TariffCard.vue";
import TariffModal from "../components/Config/TariffModal.vue";
import TelemetryModal from "../components/Config/TelemetryModal.vue";
import ExperimentalModal from "../components/Config/ExperimentalModal.vue";
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
	LoadpointType,
	Timeout,
	VehicleOption,
	MeterType,
	TariffType,
	SiteConfig,
	DeviceType,
	Notification,
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
		MessagingModal,
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
		TariffCard,
		TariffModal,
		TelemetryModal,
		ExperimentalModal,
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
			vehicles: [] as ConfigVehicle[],
			meters: [] as ConfigMeter[],
			loadpoints: [] as ConfigLoadpoint[],
			chargers: [] as ConfigCharger[],
			circuits: [] as ConfigCircuit[],
			tariffs: [] as any[], // ConfigTariff[] - tariff device entities
			tariffRefs: {
				currency: "EUR",
				grid: "",
				feedin: "",
				co2: "",
				planner: "",
				solar: [] as string[],
			},
			selectedTariffType: null as TariffType | null,
			selectedTariffTypeChoices: [] as TariffType[],
			selectedVehicleId: undefined as number | undefined,
			selectedMeterId: undefined as number | undefined,
			selectedMeterType: undefined as MeterType | undefined,
			selectedMeterTypeChoices: [] as MeterType[],
			selectedChargerId: undefined as number | undefined,
			selectedLoadpointId: undefined as number | undefined,
			selectedLoadpointType: undefined as LoadpointType | undefined,
			selectedTariffId: undefined as number | undefined,
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
		selectedMeterName() {
			return this.getMeterById(this.selectedMeterId)?.name;
		},
		gridTariff() {
			const name = this.tariffRefs?.grid;
			return name ? this.tariffs.find((t) => t.name === name) : null;
		},
		feedinTariff() {
			const name = this.tariffRefs?.feedin;
			return name ? this.tariffs.find((t) => t.name === name) : null;
		},
		co2Tariff() {
			const name = this.tariffRefs?.co2;
			return name ? this.tariffs.find((t) => t.name === name) : null;
		},
		plannerTariff() {
			const name = this.tariffRefs?.planner;
			return name ? this.tariffs.find((t) => t.name === name) : null;
		},
		solarTariffs() {
			const names = this.tariffRefs?.solar || [];
			return names.map((name) => this.tariffs.find((t) => t.name === name)).filter(Boolean);
		},
		selectedChargerName() {
			return this.getChargerById(this.selectedChargerId)?.name;
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
		shmTags(): DeviceTags {
			const { vendorId, deviceId } = store.state?.shm || {};
			// TODO: use incoming SEMP connections to determin configured/active status
			const value = !!vendorId || !!deviceId;
			return { configured: { value } };
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
			// @ts-expect-error: telemetry property exists but not in TypeScript definitions
			return store.state?.telemetry === true;
		},
		eebusTags(): DeviceTags {
			return { configured: { value: store.state?.eebus || false } };
		},
		ocppTags(): DeviceTags {
			const ocpp = store.state?.ocpp;
			const stations = ocpp?.status?.stations || [];
			if (stations.length === 0) {
				return { configured: { value: false } };
			}

			const connected = stations.filter((s) => s.status === "connected").length;
			const configured = stations.filter((s) => s.status === "configured").length;
			const detected = stations.filter((s) => s.status === "unknown").length;
			const total = connected + configured;

			const tags: Record<string, any> = {
				connections: { value: `${connected}/${total}` },
			};

			if (detected > 0) {
				tags["detected"] = { value: detected };
			}

			return tags;
		},
		modbusproxyTags(): DeviceTags {
			const config = store.state?.modbusproxy || [];
			if (config.length > 0) {
				return { amount: { value: config.length } };
			}
			return { configured: { value: false } };
		},
		messagingTags(): DeviceTags {
			return { configured: { value: store.state?.messaging || false } };
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
			await this.loadVehicles();
			await this.loadMeters();
			await this.loadSite();
			await this.loadChargers();
			await this.loadLoadpoints();
			await this.loadCircuits();
			await this.loadTariffs();
			await this.loadTariffRefs();
			await this.loadDirty();
			this.updateValues();
		},
		async loadDirty() {
			const response = await api.get("/config/dirty");
			if (response.data) {
				restart.restartNeeded = true;
			}
		},
		async loadConfig(path: string) {
			const validateStatus = (code: number) => [200, 404].includes(code);
			const response = await api.get(`/config/${path}`, { validateStatus });
			return response.status === 200 ? response.data : undefined;
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
		async loadTariffs() {
			this.tariffs = (await this.loadConfig("devices/tariff")) || [];
		},
		async loadTariffRefs() {
			const response = await api.get("/config/tariffs", {
				validateStatus: (code: number) => [200, 404].includes(code),
			});
			if (response.status === 200) {
				this.tariffRefs = response.data;
			}
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
		editMeter(type: MeterType, id?: number) {
			this.selectedMeterType = type;
			this.selectedMeterId = id;
			this.$nextTick(() => this.meterModal().show());
		},
		newMeter(type: MeterType) {
			this.selectedMeterId = undefined;
			this.selectedMeterType = type;
			this.$nextTick(() => this.meterModal().show());
		},
		addSolarBatteryMeter() {
			this.selectedMeterId = undefined;
			this.selectedMeterType = undefined;
			this.selectedMeterTypeChoices = ["pv", "battery"] as MeterType[];
			this.$nextTick(() => this.meterModal().show());
		},
		newAdditionalMeter() {
			this.selectedMeterId = undefined;
			this.selectedMeterType = undefined;
			this.selectedMeterTypeChoices = ["aux", "ext"] as MeterType[];
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
		editTariff(tariffType: TariffType, id: number) {
			this.selectedTariffId = id;
			this.selectedTariffType = tariffType;
			this.$nextTick(() => this.tariffModal().show());
		},
		newTariff(type: TariffType | null, typeChoices: TariffType[] = []) {
			this.selectedTariffId = undefined;
			this.selectedTariffType = type;
			this.selectedTariffTypeChoices = typeChoices;
			this.$nextTick(() => this.tariffModal().show());
		},
		async tariffAdded(usage: TariffType, name: string) {
			// Auto-assign tariff based on usage type
			if (usage === "grid") {
				this.tariffRefs.grid = name;
			} else if (usage === "feedin") {
				this.tariffRefs.feedin = name;
			} else if (usage === "co2") {
				this.tariffRefs.co2 = name;
			} else if (usage === "planner") {
				this.tariffRefs.planner = name;
			} else if (usage === "solar") {
				if (!this.tariffRefs.solar) this.tariffRefs.solar = [];
				this.tariffRefs.solar.push(name);
			}
			await this.saveTariffRefs(usage);
			this.tariffChanged();
		},
		async tariffRemoved(usage: TariffType) {
			// Clear assignment when tariff device is removed
			if (usage === "grid") {
				this.tariffRefs.grid = "";
				await this.saveTariffRefs("grid");
			} else if (usage === "feedin") {
				this.tariffRefs.feedin = "";
				await this.saveTariffRefs("feedin");
			} else if (usage === "co2") {
				this.tariffRefs.co2 = "";
				await this.saveTariffRefs("co2");
			} else if (usage === "planner") {
				this.tariffRefs.planner = "";
				await this.saveTariffRefs("planner");
			} else if (usage === "solar") {
				// For solar, reload assignments to get updated list
				await this.loadTariffRefs();
			}
			this.tariffChanged();
		},
		async tariffChanged() {
			this.selectedTariffId = undefined;
			this.selectedTariffType = null;
			this.tariffModal().hide();
			await this.loadTariffs();
			await this.loadTariffRefs();
			this.loadDirty();
		},
		async saveTariffRefs(key: TariffType) {
			const body = key
				? { [key]: this.tariffRefs[key as keyof typeof this.tariffRefs] }
				: this.tariffRefs;
			await api.put("/config/tariffs", body);
			await this.loadTariffRefs();
			await this.loadDirty();
		},
		tariffModal(): Modal {
			const elem = document.getElementById("tariffModal");
			if (!elem) throw new Error("tariffModal element not found");
			return Modal.getOrCreateInstance(elem);
		},
		siteChanged() {
			this.loadDirty();
		},
		yamlChanged() {
			this.loadDirty();
		},
		meterAdded(type: MeterType, name: string) {
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
		meterRemoved(type: MeterType) {
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
