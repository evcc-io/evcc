<template>
	<div class="root safe-area-inset">
		<div class="container px-4">
			<TopHeader
				ref="header"
				:title="headerTitle"
				:notifications="notifications"
				:parent-title="headerParentTitle"
				@back="goBack"
			/>
			<div class="wrapper position-relative mb-3">
				<AuthSuccessBanner
					v-if="callbackCompleted || callbackError"
					:provider-id="callbackCompleted"
					:error="callbackError"
					:auth-providers="authProviders"
				/>

				<Transition name="fade-swap-left">
					<div v-if="!mobile || !activeSlug">
						<WelcomeBanner v-if="setupRequired" class="my-4" />
						<ConfigSectionNav
							v-if="mobile"
							class="my-4"
							:sections="sectionEntries"
							@open="openSection"
						/>
					</div>
				</Transition>

				<ConfigSection v-bind="sectionProps('general')">
					<GeneralConfig
						class="box-pull-out"
						:experimental="experimental"
						:sponsor-error="hasClassError('sponsorship')"
						@site-changed="siteChanged"
					/>
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('loadpoints')">
					<div class="p-0 config-list box-pull-out">
						<DeviceCard
							v-for="loadpoint in loadpoints"
							:id="`loadpoint_${loadpoint.name}`"
							:key="loadpoint.name"
							:title="loadpoint.title"
							:name="loadpoint.name"
							:editable="!!loadpoint.id"
							:error="loadpointError(loadpoint)"
							:disabled="!!loadpoint.disable"
							data-testid="loadpoint"
							@edit="openModal('loadpoint', { id: loadpoint.id })"
							@enable="handleDisable('loadpoint', loadpoint.id!, false)"
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
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('vehicles')">
					<div class="p-0 config-list box-pull-out">
						<DeviceCard
							v-for="vehicle in vehicles"
							:id="`vehicle_${vehicle.name}`"
							:key="vehicle.name"
							:title="vehicle.config?.title || vehicle.name"
							:name="vehicle.name"
							:editable="vehicle.id >= 0"
							:error="hasDeviceError('vehicle', vehicle.name)"
							:disabled="!!vehicle.deviceDisable"
							data-testid="vehicle"
							@edit="openModal('vehicle', { id: vehicle.id })"
							@enable="handleDisable('vehicle', vehicle.id, false)"
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
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('consumers')">
					<div class="p-0 config-list box-pull-out">
						<MeterCard
							v-for="meter in consumerMeters"
							:key="meter.name"
							:meter="meter"
							meter-type="consumer"
							:has-error="hasDeviceError('meter', meter.name)"
							:tags="deviceTags('meter', meter.name)"
							@edit="(type, id) => openModal('meter', { type, id })"
							@enable="handleDisable('meter', meter.id, false)"
						/>
						<MeterCard
							v-for="meter in auxMeters"
							:key="meter.name"
							:meter="meter"
							meter-type="aux"
							:has-error="hasDeviceError('meter', meter.name)"
							:tags="deviceTags('meter', meter.name)"
							@edit="(type, id) => openModal('meter', { type, id })"
							@enable="handleDisable('meter', meter.id, false)"
						/>
						<NewDeviceButton
							data-testid="add-consumer"
							:title="$t('config.main.addConsumer')"
							@click="openModal('meter', { choices: ['consumer', 'aux'] })"
						/>
					</div>
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('grid')">
					<div class="p-0 config-list box-pull-out">
						<MeterCard
							v-if="gridMeter"
							:meter="gridMeter"
							:title="$t('config.grid.title')"
							meter-type="grid"
							:has-error="hasDeviceError('meter', gridMeter.name)"
							:tags="deviceTags('meter', gridMeter.name)"
							@edit="(type, id) => openModal('meter', { type, id })"
							@enable="handleDisable('meter', gridMeter.id, false)"
						/>
						<NewDeviceButton
							v-else
							:title="$t('config.main.addGrid')"
							data-testid="add-grid"
							@click="openModal('meter', { type: 'grid' })"
						/>
						<DeviceCard
							v-for="curtailer in curtailerDevices"
							:id="`curtailer_${curtailer.name}`"
							:key="curtailer.name"
							:name="curtailer.name"
							:title="curtailerTitle(curtailer)"
							:editable="!!curtailer.id"
							:error="hasDeviceError('curtailer', curtailer.name)"
							:banner="curtailmentBanner('curtailer', curtailer.name)"
							data-testid="curtailer"
							@edit="openModal('curtailer', { id: curtailer.id })"
						>
							<template #icon>
								<CurtailIcon />
							</template>
							<template #tags>
								<DeviceTags :tags="deviceTags('curtailer', curtailer.name)" />
							</template>
						</DeviceCard>
						<NewDeviceButton
							v-if="pvMeters.length > 0"
							:title="$t('config.main.addCurtailer')"
							data-testid="add-curtailer"
							@click="openModal('curtailer')"
						/>
					</div>
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('pv-battery')">
					<div class="p-0 config-list box-pull-out">
						<MeterCard
							v-for="meter in pvMeters"
							:key="meter.name"
							:meter="meter"
							meter-type="pv"
							:has-error="hasDeviceError('meter', meter.name)"
							:tags="deviceTags('meter', meter.name)"
							:banner="curtailmentBanner('meter', meter.name)"
							@edit="(type, id) => openModal('meter', { type, id })"
							@enable="handleDisable('meter', meter.id, false)"
						/>
						<MeterCard
							v-for="meter in batteryMeters"
							:key="meter.name"
							:meter="meter"
							meter-type="battery"
							:has-error="hasDeviceError('meter', meter.name)"
							:tags="deviceTags('meter', meter.name)"
							@edit="(type, id) => openModal('meter', { type, id })"
							@enable="handleDisable('meter', meter.id, false)"
						/>
						<NewDeviceButton
							:title="$t('config.main.addPvBattery')"
							@click="openModal('meter', { choices: ['pv', 'battery'] })"
						/>
					</div>
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('meters')">
					<div class="p-0 config-list box-pull-out">
						<MeterCard
							v-for="meter in extMeters"
							:key="meter.name"
							:meter="meter"
							meter-type="ext"
							:has-error="hasDeviceError('meter', meter.name)"
							:tags="deviceTags('meter', meter.name)"
							@edit="(type, id) => openModal('meter', { type, id })"
							@enable="handleDisable('meter', meter.id, false)"
						/>
						<NewDeviceButton
							data-testid="add-additional"
							:title="$t('config.main.addAdditional')"
							@click="openModal('meter', { type: 'ext' })"
						/>
					</div>
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('tariffs')">
					<div v-if="!!tariffsYamlSource" class="p-0 config-list box-pull-out">
						<DeviceCard
							:title="$t('config.tariff.title')"
							:editable="tariffsYamlSource === 'db'"
							:unconfigured="isUnconfigured(tariffTags)"
							:error="hasClassError('tariff')"
							:badge="tariffsYamlSource === 'db'"
							data-testid="tariffs-legacy"
							:currency="currency"
							@edit="openModal('tariffsLegacy')"
						>
							<template #icon>
								<shopicon-regular-receivepayment></shopicon-regular-receivepayment>
							</template>
							<template #tags>
								<DeviceTags :tags="tariffTags" :currency="currency" />
							</template>
						</DeviceCard>
					</div>
					<div v-else class="p-0 config-list box-pull-out">
						<TariffCard
							v-if="gridTariff"
							:tariff="gridTariff"
							tariff-type="grid"
							:has-error="hasDeviceError('tariff', gridTariff.name)"
							:tags="deviceTags('tariff', gridTariff.name)"
							:currency="currency"
							@edit="openModal('tariff', { type: 'grid', id: gridTariff.id })"
							@enable="handleDisable('tariff', gridTariff.id, false)"
						/>
						<TariffCard
							v-if="feedInTariff"
							:tariff="feedInTariff"
							tariff-type="feedIn"
							:has-error="hasDeviceError('tariff', feedInTariff.name)"
							:tags="deviceTags('tariff', feedInTariff.name)"
							:currency="currency"
							@edit="openModal('tariff', { type: 'feedIn', id: feedInTariff.id })"
							@enable="handleDisable('tariff', feedInTariff.id, false)"
						/>
						<NewDeviceButton
							v-if="possibleTariffTypes.length"
							:title="$t('config.tariff.addTariff')"
							@click="openModal('tariff', { choices: possibleTariffTypes })"
						/>
						<TariffCard
							v-if="co2Tariff"
							:tariff="co2Tariff"
							tariff-type="co2"
							:has-error="hasDeviceError('tariff', co2Tariff.name)"
							:tags="deviceTags('tariff', co2Tariff.name)"
							@edit="openModal('tariff', { type: 'co2', id: co2Tariff.id })"
							@enable="handleDisable('tariff', co2Tariff.id, false)"
						/>
						<TariffCard
							v-for="tariff in solarTariffs"
							:key="tariff.name"
							:tariff="tariff"
							tariff-type="solar"
							:has-error="hasDeviceError('tariff', tariff.name)"
							:tags="deviceTags('tariff', tariff.name)"
							:currency="currency"
							@edit="openModal('tariff', { type: 'solar', id: tariff.id })"
							@enable="handleDisable('tariff', tariff.id, false)"
						/>
						<TariffCard
							v-if="temperatureTariff"
							:tariff="temperatureTariff"
							tariff-type="temperature"
							:has-error="hasDeviceError('tariff', temperatureTariff.name)"
							:tags="deviceTags('tariff', temperatureTariff.name)"
							@edit="
								openModal('tariff', {
									type: 'temperature',
									id: temperatureTariff.id,
								})
							"
						/>
						<TariffCard
							v-if="plannerTariff"
							:tariff="plannerTariff"
							tariff-type="planner"
							:has-error="hasDeviceError('tariff', plannerTariff.name)"
							:tags="deviceTags('tariff', plannerTariff.name)"
							:currency="currency"
							@edit="openModal('tariff', { type: 'planner', id: plannerTariff.id })"
							@enable="handleDisable('tariff', plannerTariff.id, false)"
						/>
						<NewDeviceButton
							v-if="possibleForecastTypes.length"
							:title="$t('config.tariff.addForecast')"
							@click="openModal('tariff', { choices: possibleForecastTypes })"
						/>
					</div>
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('integrations')">
					<div class="p-0 config-list box-pull-out">
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
							:unconfigured="isUnconfigured(circuitsTags)"
							:banner="
								hemsDimmed && circuitsRoot
									? $t('config.deviceValue.dimmed')
									: undefined
							"
							data-testid="circuits"
							@edit="openModal('circuits')"
						>
							<template #icon><CircuitsIcon /></template>
							<template #tags>
								<DeviceTags v-if="!circuitsRoot" :tags="circuitsTags" />
								<CircuitTags v-else :nodes="[circuitsRoot]" />
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
								<p v-if="hemsLabel" class="my-2 fw-bold">{{ hemsLabel }}</p>
								<DeviceTags :tags="hemsTags" />
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
							:title="$t('config.remote.title')"
							editable
							:unconfigured="isUnconfigured(remoteTags)"
							data-testid="remote-access"
							@edit="openModal('remote')"
						>
							<template #icon><RemoteAccessIcon /></template>
							<template #tags>
								<DeviceTags :tags="remoteTags" />
							</template>
						</DeviceCard>
						<DeviceCard
							v-if="experimental"
							:title="`${$t('config.optimizer.title')} 🧪`"
							editable
							:unconfigured="isUnconfigured(optimizerTags)"
							data-testid="optimizer"
							@edit="openModal('optimizer')"
						>
							<template #icon><OptimizerIcon /></template>
							<template #tags>
								<DeviceTags :tags="optimizerTags" />
							</template>
						</DeviceCard>
					</div>
				</ConfigSection>

				<ConfigSection v-bind="sectionProps('services')">
					<div class="p-0 config-list box-pull-out">
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
						<DeviceCard
							v-if="experimental"
							:title="`${$t('config.mcp.title')} 🧪`"
							editable
							data-testid="mcp"
							@edit="openModal('mcp')"
						>
							<template #icon><McpIcon /></template>
						</DeviceCard>
					</div>
				</ConfigSection>

				<hr v-if="!mobile" class="my-5" />

				<ConfigSection v-bind="sectionProps('system')">
					<div
						data-testid="config-system"
						class="round-box box-pull-out p-4 d-grid d-md-flex gap-4 flex-wrap"
					>
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
				</ConfigSection>

				<LoadpointModal
					:vehicleOptions="vehicleOptions"
					:loadpointCount="loadpoints.length"
					:chargers="chargers"
					:chargerValues="deviceValues['charger']"
					:meters="meters"
					:circuits="circuits"
					:hasDeviceError="hasDeviceError"
					@changed="loadpointChanged"
					@dismissed="loadpointDismissed"
					@disable="({ id, disable }) => handleDisable('loadpoint', id, disable)"
				/>
				<VehicleModal
					:is-sponsor="isSponsor"
					@vehicle-changed="vehicleChanged"
					@disable="({ id, disable }) => handleDisable('vehicle', id, disable)"
				/>
				<MeterModal
					:is-sponsor="isSponsor"
					@changed="meterChanged"
					@disable="({ id, disable }) => handleDisable('meter', id, disable)"
				/>
				<ChargerModal :is-sponsor="isSponsor" :ocpp="ocpp" @changed="chargerChanged" />
				<InfluxModal @changed="loadDirty" />
				<MqttModal @changed="loadDirty" />
				<NetworkModal @changed="loadDirty" />
				<ControlModal @changed="loadDirty" />
				<HemsModal
					:id="hemsDevices[0]?.id"
					:yamlSource="hems?.yamlSource"
					@changed="hemsChanged"
				/>
				<ShmModal @changed="loadDirty" />
				<MessagingLegacyModal @changed="loadDirty" />
				<MessagingModal :messengers="messengers" @changed="loadDirty" />
				<MessengerModal @changed="messengerChanged" />
				<CurtailerModal @changed="curtailerChanged" />
				<TariffsLegacyModal @changed="loadDirty" />
				<TariffModal
					:currency="currency"
					@changed="tariffChanged"
					@disable="({ id, disable }) => handleDisable('tariff', id, disable)"
				/>
				<TelemetryModal :is-sponsor="isSponsor" :telemetry="telemetry" />
				<OptimizerModal :is-sponsor="isSponsor" />
				<McpModal />
				<ExperimentalModal :experimental="experimental" />
				<RemoteModal :remote="remote" :is-sponsor="isSponsor" :site-title="siteTitle" />
				<TitleModal @changed="loadDirty" />
				<ModbusProxyModal :is-sponsor="isSponsor" @changed="loadDirty" />
				<CircuitsModal :gridMeter="gridMeter" :extMeters="extMeters" @changed="loadDirty" />
				<EebusModal
					:status="eebus?.status"
					:yamlSource="eebus?.yamlSource"
					@changed="loadDirty"
				/>
				<OcppModal :ocpp="ocpp" :stationTitles="stationTitles" />
				<OcppForwarderModal @changed="loadDirty" />
				<BackupRestoreModal v-bind="backupRestoreProps" />
				<SecurityModal :auth-disabled="authDisabled" />
				<ApiKeyModal :auth-disabled="authDisabled" />
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
import "@h2d2/shopicons/es/regular/settings";
import "@h2d2/shopicons/es/regular/car3";
import NewDeviceButton from "../components/Config/NewDeviceButton.vue";
import api from "../api";
import listDetail from "../mixins/listDetail";
import ChargerModal from "../components/Config/ChargerModal.vue";
import ConfigSection from "../components/Config/ConfigSection.vue";
import ConfigSectionNav, { type SectionEntry } from "../components/Config/ConfigSectionNav.vue";
import IntegrationsIcon from "../components/MaterialIcon/Integrations.vue";
import ServicesIcon from "../components/MaterialIcon/Services.vue";
import SystemIcon from "../components/MaterialIcon/System.vue";
import MeterIcon from "../components/VehicleIcon/Meter.vue";
import GenericIcon from "../components/VehicleIcon/Generic.vue";
import CircuitsIcon from "../components/MaterialIcon/Circuits.vue";
import CircuitsModal from "../components/Config/CircuitsModal.vue";
import CircuitTags from "../components/Config/CircuitTags.vue";
import collector from "../mixins/collector";
import ControlModal from "../components/Config/ControlModal.vue";
import CurtailerModal from "../components/Config/CurtailerModal.vue";
import CurtailIcon from "../components/MaterialIcon/Curtail.vue";
import DeviceCard from "../components/Config/DeviceCard.vue";
import DeviceTags from "../components/Config/DeviceTags.vue";
import EebusIcon from "../components/MaterialIcon/Eebus.vue";
import EebusModal from "../components/Config/EebusModal.vue";
import OcppIcon from "../components/MaterialIcon/Ocpp.vue";
import OcppModal from "../components/Config/OcppModal.vue";
import OcppForwarderModal from "../components/Config/OcppForwarderModal.vue";
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
import MessengerModal from "@/components/Config/Messaging/MessengerModal.vue";
import MessagingLegacyModal from "@/components/Config/Messaging/MessagingLegacyModal.vue";
import MeterModal from "../components/Config/MeterModal.vue";
import MeterCard from "../components/Config/MeterCard.vue";
import { createDeviceUtils } from "../components/Config/DeviceModal";
import { openModal, type ModalResult } from "@/configModal";
import ModbusProxyIcon from "../components/MaterialIcon/ModbusProxy.vue";
import ModbusProxyModal from "../components/Config/ModbusProxyModal.vue";
import MqttIcon from "../components/MaterialIcon/Mqtt.vue";
import MqttModal from "../components/Config/MqttModal.vue";
import RemoteAccessIcon from "../components/MaterialIcon/RemoteAccess.vue";
import RemoteModal from "../components/Config/Remote/RemoteModal.vue";
import { isRemoteClientActive } from "@/utils/remote";
import NetworkModal from "../components/Config/NetworkModal.vue";
import NotificationIcon from "../components/MaterialIcon/Notification.vue";
import OptimizerIcon from "../components/MaterialIcon/Optimizer.vue";
import OptimizerModal from "../components/Config/OptimizerModal.vue";
import McpIcon from "../components/MaterialIcon/Mcp.vue";
import McpModal from "../components/Config/McpModal.vue";
import restart, { performRestart } from "../restart";
import SponsorModal from "../components/Config/SponsorModal.vue";
import store from "../store";
import TariffsLegacyModal from "../components/Config/TariffsLegacyModal.vue";
import TariffCard from "../components/Config/TariffCard.vue";
import TariffModal from "../components/Config/TariffModal.vue";
import TelemetryModal from "../components/Config/TelemetryModal.vue";
import ExperimentalModal from "../components/Config/ExperimentalModal.vue";
import TitleModal from "../components/Config/TitleModal.vue";
import Header from "../components/Top/Header.vue";
import VehicleIcon from "../components/VehicleIcon";
import VehicleModal from "../components/Config/VehicleModal.vue";
import { defineComponent, markRaw, type PropType } from "vue";
import type {
	ConfigCharger,
	ConfigVehicle,
	ConfigCircuit,
	ConfigCurtailer,
	ConfigMessenger,
	ConfigHems,
	ConfigLoadpoint,
	ConfigMeter,
	Timeout,
	VehicleOption,
	MeterType,
	TariffType,
	SiteConfig,
	DeviceType,
	Notification,
	Remote,
} from "@/types/evcc";
import { ConfigType, CURRENCY } from "@/types/evcc";
import { circuitTree, type CircuitNode } from "@/utils/circuits";

type DeviceValuesMap = Record<DeviceType, Record<string, any>>;

// section slug (anchor/deep link) -> title i18n key, in display order
const SECTION_TITLES: Record<string, string> = {
	general: "config.section.general",
	loadpoints: "config.section.loadpoints",
	vehicles: "config.section.vehicles",
	consumers: "config.section.consumers",
	grid: "config.section.grid",
	"pv-battery": "config.section.meter",
	meters: "config.section.additionalMeter",
	tariffs: "config.tariff.title",
	integrations: "config.section.integrations",
	services: "config.section.services",
	system: "config.section.system",
};

type DeviceTags = Record<
	string,
	{ value?: any; error?: boolean; warning?: boolean; muted?: boolean; options?: any }
>;

import BackupRestoreModal from "@/components/Config/BackupRestoreModal.vue";
import WelcomeBanner from "../components/Config/WelcomeBanner.vue";
import AuthSuccessBanner from "../components/Config/AuthSuccessBanner.vue";
import PasswordModal from "../components/Auth/PasswordModal.vue";
import SecurityModal from "../components/Config/Security/SecurityModal.vue";
import ApiKeyModal from "../components/Config/Security/ApiKeyModal.vue";
import AuthProvidersCard from "../components/Config/AuthProvidersCard.vue";

export default defineComponent({
	name: "Config",
	components: {
		NewDeviceButton,
		BackupRestoreModal,
		ChargerModal,
		ConfigSection,
		ConfigSectionNav,
		CircuitsIcon,
		CircuitsModal,
		CircuitTags,
		ControlModal,
		CurtailerModal,
		CurtailIcon,
		DeviceCard,
		DeviceTags,
		EebusIcon,
		EebusModal,
		OcppIcon,
		OcppModal,
		OcppForwarderModal,
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
		RemoteAccessIcon,
		RemoteModal,
		NetworkModal,
		NotificationIcon,
		OptimizerIcon,
		OptimizerModal,
		McpIcon,
		McpModal,
		SponsorModal,
		TariffsLegacyModal,
		TariffCard,
		TariffModal,
		TelemetryModal,
		ExperimentalModal,
		TitleModal,
		TopHeader: Header,
		VehicleIcon,
		VehicleModal,
		WelcomeBanner,
		AuthSuccessBanner,
		PasswordModal,
		SecurityModal,
		ApiKeyModal,
		AuthProvidersCard,
	},
	mixins: [formatter, collector, listDetail],
	props: {
		offline: Boolean,
		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
	},
	data() {
		return {
			messengers: [] as ConfigMessenger[],
			curtailers: [] as ConfigCurtailer[],
			vehicles: [] as ConfigVehicle[],
			meters: [] as ConfigMeter[],
			loadpoints: [] as ConfigLoadpoint[],
			chargers: [] as ConfigCharger[],
			circuits: [] as ConfigCircuit[],
			hemsDevices: [] as ConfigHems[],
			tariffs: [] as any[], // ConfigTariff[] - tariff device entities
			tariffRefs: {
				grid: "",
				feedIn: "",
				co2: "",
				planner: "",
				solar: [] as string[],
				temperature: "",
			},
			site: {
				grid: "",
				pv: [] as string[],
				battery: [] as string[],
				title: "",
				aux: null as string[] | null,
				ext: null as string[] | null,
				consumer: null as string[] | null,
				curtail: null as string[] | null,
			} as SiteConfig,
			deviceValueTimeout: null as Timeout,
			deviceValues: {
				meter: {},
				vehicle: {},
				charger: {},
				loadpoint: {},
				messenger: {},
				tariff: {},
				curtailer: {},
			} as DeviceValuesMap,
			isComponentMounted: true,
			isPageVisible: true,
		};
	},
	head() {
		return { title: this.$t("config.main.title") };
	},
	computed: {
		activeSlug(): string | undefined {
			const slug = this.$route.hash.slice(1);
			return SECTION_TITLES[slug] ? slug : undefined;
		},
		headerTitle(): string {
			return this.mobile && this.activeSlug
				? this.$t(SECTION_TITLES[this.activeSlug]!)
				: this.$t("config.main.title");
		},
		headerParentTitle(): string {
			return this.mobile && this.activeSlug ? this.$t("config.main.title") : "";
		},
		sectionEntries(): SectionEntry[] {
			const meterError = (meters: ConfigMeter[]) =>
				meters.some((m) => this.hasDeviceError("meter", m.name));
			const meterDisabled = (meters: ConfigMeter[]) => meters.some((m) => m.deviceDisable);
			const auxAndConsumer = [...this.consumerMeters, ...this.auxMeters];
			const pvAndBattery = [...this.pvMeters, ...this.batteryMeters];
			const configuredTariffCount =
				[
					this.gridTariff,
					this.feedInTariff,
					this.co2Tariff,
					this.temperatureTariff,
					this.plannerTariff,
				].filter(Boolean).length + this.solarTariffs.length;
			// key = config.<key>.title i18n stem and fatal error class (see errorClass exceptions)
			const integrationTags: Record<string, DeviceTags> = {
				mqtt: this.mqttTags,
				messaging: this.messagingTags,
				influx: this.influxTags,
				circuits: this.circuitsTags,
				hems: this.hemsTags,
				modbusproxy: this.modbusproxyTags,
				remote: this.remoteTags,
				optimizer: this.optimizerTags,
			};
			const errorClass: Record<string, string> = {
				messaging: "messenger",
				circuits: "circuit",
			};
			const integrations = Object.entries(integrationTags).map(([key, tags]) => ({
				key,
				error: errorClass[key] || key,
				configured: !this.isUnconfigured(tags),
			}));
			const entries: Omit<SectionEntry, "title">[] = [
				{
					slug: "general",
					icon: "shopicon-regular-settings",
					subline: this.siteTitle || undefined,
					error: this.hasClassError("sponsorship"),
					warning: !!this.sponsor?.status?.expiresSoon,
				},
				{
					slug: "loadpoints",
					icon: markRaw(LoadpointIcon),
					count: this.loadpoints.length,
					error: this.loadpoints.some((lp) => this.loadpointError(lp)),
					warning: this.loadpoints.some((lp) => lp.disable),
				},
				{
					slug: "vehicles",
					icon: "shopicon-regular-car3",
					count: this.vehicles.length,
					error: this.vehicles.some((v) => this.hasDeviceError("vehicle", v.name)),
					warning: this.vehicles.some((v) => v.deviceDisable),
				},
				{
					slug: "consumers",
					icon: markRaw(GenericIcon),
					count: auxAndConsumer.length,
					error: meterError(auxAndConsumer),
					warning: meterDisabled(auxAndConsumer),
				},
				{
					slug: "grid",
					icon: "shopicon-regular-powersupply",
					count: (this.gridMeter ? 1 : 0) + this.curtailerDevices.length,
					error:
						(!!this.gridMeter && this.hasDeviceError("meter", this.gridMeter.name)) ||
						this.curtailerDevices.some((c) => this.hasDeviceError("curtailer", c.name)),
				},
				{
					slug: "pv-battery",
					icon: "shopicon-regular-sun",
					count: pvAndBattery.length,
					error: meterError(pvAndBattery),
					warning:
						this.pvMeters.some((m) => this.curtailmentBanner("meter", m.name)) ||
						meterDisabled(pvAndBattery),
				},
				{
					slug: "meters",
					icon: markRaw(MeterIcon),
					count: this.extMeters.length,
					error: meterError(this.extMeters),
					warning: meterDisabled(this.extMeters),
				},
				{
					slug: "tariffs",
					icon: "shopicon-regular-receivepayment",
					count: this.tariffsYamlSource ? 1 : configuredTariffCount,
					error:
						this.hasClassError("tariff") ||
						this.tariffs.some((t) => this.hasDeviceError("tariff", t.name)),
				},
				{
					slug: "integrations",
					icon: markRaw(IntegrationsIcon),
					subline:
						integrations
							.filter((i) => i.configured)
							.map((i) => this.$t(`config.${i.key}.title`))
							.join(" · ") || undefined,
					error: integrations.some((i) => this.hasClassError(i.error)),
					warning: Object.values(this.authProviders || {}).some((p) => !p.authenticated),
				},
				{
					slug: "services",
					icon: markRaw(ServicesIcon),
					error: ["ocpp", "shm", "eebus", "mcp"].some((c) => this.hasClassError(c)),
				},
				{
					slug: "system",
					icon: markRaw(SystemIcon),
				},
			];
			return entries.map((e) => ({ ...e, title: this.$t(SECTION_TITLES[e.slug]!) }));
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
		currency(): CURRENCY {
			return store.state?.currency ?? CURRENCY.EUR;
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
		curtailerDevices(): ConfigCurtailer[] {
			const names = this.site?.curtail || [];
			return names
				.map((name) => this.curtailers.find((c) => c.name === name))
				.filter((c): c is ConfigCurtailer => c !== undefined);
		},
		consumerMeters() {
			return this.getMetersByNames(this.site?.consumer);
		},
		gridTariff() {
			const name = this.tariffRefs?.grid;
			return name ? this.tariffs.find((t) => t.name === name) : null;
		},
		feedInTariff() {
			const name = this.tariffRefs?.feedIn;
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
		temperatureTariff() {
			const name = this.tariffRefs?.temperature;
			return name ? this.tariffs.find((t) => t.name === name) : null;
		},
		solarTariffs() {
			const names = this.tariffRefs?.solar || [];
			return names.map((name) => this.tariffs.find((t) => t.name === name)).filter(Boolean);
		},
		possibleTariffTypes(): TariffType[] {
			const types: TariffType[] = [];
			if (!this.gridTariff) types.push("grid");
			if (!this.feedInTariff) types.push("feedIn");
			return types;
		},
		possibleForecastTypes(): TariffType[] {
			const types: TariffType[] = [];
			if (!this.co2Tariff) types.push("co2");
			types.push("solar"); // Solar can have multiple
			if (!this.temperatureTariff) types.push("temperature");
			if (!this.plannerTariff) types.push("planner");
			return types;
		},
		tariffTags(): DeviceTags {
			const { tariffGrid, tariffFeedIn, tariffCo2, tariffSolar, tariffTemperature } =
				store.state;
			if (
				tariffGrid === undefined &&
				tariffFeedIn === undefined &&
				tariffCo2 === undefined &&
				tariffSolar === undefined &&
				tariffTemperature === undefined
			) {
				return { configured: { value: false } };
			}
			const tags = {
				gridPrice: {},
				feedinPrice: {},
				co2: {},
				solarForecast: {},
				outdoorTemp: {},
			};
			if (tariffGrid) {
				tags.gridPrice = { value: tariffGrid };
			}
			if (tariffFeedIn) {
				tags.feedinPrice = { value: tariffFeedIn * -1 };
			}
			if (tariffCo2) {
				tags.co2 = { value: tariffCo2 };
			}
			if (tariffSolar) {
				tags.solarForecast = { value: tariffSolar };
			}
			if (tariffTemperature !== undefined) {
				tags.outdoorTemp = { value: tariffTemperature };
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
			return this.vehicles
				.filter((v) => !v.deviceDisable)
				.map((v) => ({ key: v.name, name: v.config?.title || v.name }));
		},
		hems() {
			return store.state?.hems;
		},
		hemsTags(): DeviceTags {
			const result: DeviceTags = {};
			const exportLimit = store.state?.gridExportLimit || 0;
			if (exportLimit > 0) {
				result["exportLimit"] = { value: exportLimit };
			}
			if (this.hemsDevices.length === 0 && !this.hems?.config?.configured) {
				return exportLimit > 0 ? result : { configured: { value: false } };
			}
			const status = store.state?.hems?.status;
			if (!status) {
				return { ...result, configured: { value: true } };
			}
			if (status.dimmed && status.maxConsumptionPower) {
				result["dimLimit"] = {
					value: status.maxConsumptionPower,
					warning: true,
				};
			} else if (status.dimmed !== undefined) {
				result["dimmed"] = { value: status.dimmed };
			}
			if ((status.curtailed ?? 100) < 100 && status.maxProductionPower !== undefined) {
				result["curtailLimit"] = {
					value: status.maxProductionPower,
					warning: true,
				};
			} else if (status.curtailed !== undefined) {
				result["curtailed"] = { value: status.curtailed < 100 };
			}

			return result;
		},
		hemsLabel(): string {
			const dev = this.hemsDevices[0];
			if (!dev) return "";
			if (dev.deviceProduct) return dev.deviceProduct;
			if (dev.type === ConfigType.Custom) return this.$t("config.hems.customOption");
			return "";
		},
		remote(): Remote | undefined {
			return store.state?.remote;
		},
		remoteTags(): DeviceTags {
			const remote = this.remote;
			if (!remote?.status?.url) {
				return { configured: { value: false } };
			}
			const tags: DeviceTags = {
				remoteEnabled: { value: remote.config?.enabled },
				connected: {
					value: remote.status?.connected,
					error: remote.config?.enabled && !remote.status?.connected,
				},
			};
			if (remote.status?.loginBlocked) {
				tags["loginBlocked"] = { value: true, error: true };
			}
			if (remote.config?.enabled) {
				const lastSeen = remote.status?.lastSeen;
				const count = lastSeen
					? Object.keys(lastSeen).filter((u) => isRemoteClientActive(lastSeen, u)).length
					: 0;
				tags["activeClients"] = { value: count };
			}
			return tags;
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
		optimizerTags(): DeviceTags {
			if (!store.state?.optimizer) return { configured: { value: false } };
			return { configured: { value: true } };
		},
		modbusproxyTags(): DeviceTags {
			const config = store.state?.modbusproxy || [];
			if (config.length > 0) {
				return { amount: { value: config.length } };
			}
			return { configured: { value: false } };
		},
		circuitsTags(): DeviceTags {
			return this.circuitsRoot ? {} : { configured: { value: false } };
		},
		// maps an OCPP station id to its loadpoint title (fallback: charger title)
		stationTitles(): Record<string, string> {
			const map: Record<string, string> = {};
			this.chargers.forEach((charger) => {
				const stationId = charger.config?.["stationid"];
				if (typeof stationId !== "string" || !stationId) return;
				const loadpoint = this.loadpoints.find((lp) => lp.charger === charger.name);
				const title = loadpoint?.title || charger.config?.title;
				if (title) map[stationId] = title;
			});
			return map;
		},
		messagingTags(): DeviceTags {
			if (this.messagingUiConfigured) {
				const events = store.state?.messagingEvents || [];
				const enabledEvents = Object.values(events).filter((e: any) => !e.disabled).length;
				return {
					events: { value: enabledEvents },
					messengers: { value: this.messengers.length },
				};
			}
			return { configured: { value: this.messagingYamlConfigured } };
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
				Object.values(store.state.messagingEvents ?? {}).some((e) => !e.disabled)
			);
		},
		authDisabled() {
			return store.state?.authDisabled || false;
		},
		backupRestoreProps() {
			return {
				authDisabled: this.authDisabled,
			};
		},
		circuitsRoot(): CircuitNode | null {
			return circuitTree(store.state?.circuits || {});
		},
		hemsDimmed(): boolean {
			// only consumption limits matter for circuits, curtailment affects feed-in
			return !!store.state?.hems?.status?.dimmed;
		},
		tariffsYamlSource() {
			return store.state?.tariffs?.yamlSource;
		},
		tariffsUiVisible() {
			return this.tariffsYamlSource === undefined;
		},
		tariffsYamlVisible() {
			return !this.tariffsUiVisible;
		},
		tariffsYamlDisabled() {
			return this.tariffsYamlSource === "file";
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
		sectionProps(slug: string) {
			return {
				slug,
				title: this.$t(SECTION_TITLES[slug]!),
				mobile: this.mobile,
				active: this.mobile && this.activeSlug === slug,
			};
		},
		openSection(slug: string) {
			this.$router.push({ path: "/config", hash: `#${slug}` });
		},
		async handleDisable(deviceClass: DeviceType, id: number, disable: boolean) {
			const promptKey = disable
				? "config.general.confirmDisable"
				: "config.general.confirmEnable";
			if (!window.confirm(this.$t(promptKey))) return;
			const refresh: Partial<Record<DeviceType, () => void>> = {
				meter: () => this.meterChanged({ action: "updated" }),
				tariff: () => this.tariffChanged({ action: "updated" }),
				vehicle: () => this.vehicleChanged(),
				loadpoint: () => this.loadpointChanged(),
			};
			try {
				if (deviceClass === "loadpoint") {
					const { data } = await api.get(`config/loadpoints/${id}`);
					await api.put(`config/loadpoints/${id}`, { ...data, disable });
				} else {
					await createDeviceUtils(deviceClass).disable(id, disable);
				}
				refresh[deviceClass]?.();
				await this.loadDirty();
			} catch (e) {
				console.error("disable failed", e);
			}
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
			await this.loadMessengers();
			await this.loadCurtailers();
			await this.loadTariffs();
			await this.loadTariffRefs();
			await this.loadHems();
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
		async loadCurtailers() {
			this.curtailers = (await this.loadConfig("devices/curtailer")) || [];
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
		async loadHems() {
			this.hemsDevices = (await this.loadConfig("devices/hems")) || [];
		},
		async loadCircuits() {
			this.circuits = (await this.loadConfig("devices/circuit")) || [];
		},
		async loadTariffs() {
			this.tariffs = (await this.loadConfig("devices/tariff")) || [];
		},
		async loadTariffRefs() {
			const response = await api.get("/config/tariff", {
				validateStatus: (code: number) => [200, 404].includes(code),
			});
			if (response.status === 200) {
				this.tariffRefs = response.data;
				if (!this.tariffRefs.solar) this.tariffRefs.solar = [];
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
					case "consumer":
						if (!this.site.consumer) this.site.consumer = [];
						this.site.consumer.push(name);
						this.saveSite("consumer");
						break;
				}
			}

			// Converted: move ext meter to consumer (history is reconciled on restart)
			if (result.action === "converted") {
				const name = this.meters.find((m) => m.id === result.id)?.name;
				if (name) {
					const ext = (this.site.ext || []).filter((n) => n !== name);
					const consumer = [...(this.site.consumer || []), name];
					await api.put("/config/site", { ext, consumer });
					await this.loadSite();
				}
			}

			// Removed: reload site config
			if (result.action === "removed") {
				await this.loadSite();
			}

			// Reload meters and update UI
			await this.loadMeters();
			await this.loadDirty();
			this.updateValues();
		},
		async chargerChanged() {
			await this.loadChargers();
			await this.loadDirty();
			this.updateValues();
		},
		async hemsChanged() {
			await this.loadHems();
			await this.loadDirty();
		},
		async loadpointChanged() {
			await this.loadLoadpoints();
			this.loadDirty();
		},
		async loadpointDismissed() {
			await this.loadChargers();
			await this.loadMeters();
			this.updateValues();
		},
		vehicleChanged() {
			this.loadVehicles();
			this.loadDirty();
		},
		async tariffChanged(result: ModalResult) {
			if (result.action === "added") {
				const usage = result.type as TariffType;
				const name = result.name!;
				if (usage === "solar") {
					this.tariffRefs.solar.push(name);
				} else {
					this.tariffRefs[usage] = name;
				}
				await api.put("/config/tariff", this.tariffRefs);
			}
			if (result.action === "removed") {
				await this.loadTariffRefs();
			}
			await this.loadTariffs();
			await this.loadTariffRefs();
			await this.loadDirty();
			this.updateValues();
		},
		openMessagingModal() {
			const modalName = this.messagingYamlSource === "db" ? "messaginglegacy" : "messaging";
			openModal(modalName);
		},
		async messengerChanged() {
			this.loadMessengers();
			this.loadDirty();
		},
		curtailerTitle(curtailer: ConfigCurtailer): string {
			return (
				curtailer.deviceTitle ||
				curtailer.config?.template ||
				this.$t("config.curtailer.title")
			);
		},
		async curtailerChanged(result: ModalResult) {
			if (result.action === "added" && result.name) {
				this.site.curtail = [...(this.site.curtail || []), result.name];
				await this.saveSite("curtail");
			}
			if (result.action === "removed") {
				await this.loadSite();
			}
			await this.loadCurtailers();
			await this.loadDirty();
			this.updateValues();
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
					vehicle: this.vehicles,
					charger: this.chargers,
					tariff: this.tariffs,
					curtailer: this.curtailers,
				} as Record<DeviceType, any[]>;
				for (const type in devices) {
					for (const device of devices[type as DeviceType]) {
						if (device.deviceDisable) continue;
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
		curtailmentBanner(type: DeviceType, name: string): string | undefined {
			// devices report the allowed feed-in percent, 100 = uncurtailed
			const value = this.deviceTags(type, name)["curtailed"]?.value;
			return typeof value === "number" && value < 100
				? this.$t("config.deviceValue.productionLimited")
				: undefined;
		},
		loadpointTags(loadpoint: ConfigLoadpoint) {
			const { charger, meter } = loadpoint;
			const chargerTags = charger ? this.deviceTags("charger", charger) : {};
			const meterTags = meter ? this.deviceTags("meter", meter) : {};
			return { ...chargerTags, ...meterTags };
		},
		openModal,
		loadpointError(loadpoint: ConfigLoadpoint): boolean {
			return (
				this.hasDeviceError("loadpoint", loadpoint.name) ||
				this.hasDeviceError("charger", loadpoint.charger) ||
				this.hasDeviceError("meter", loadpoint.meter)
			);
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
}) as any;
</script>
<style scoped>
/* transition transforms must not make the page x-scrollable */
.container {
	overflow-x: clip;
}
.config-list {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
	grid-gap: 1rem;
	margin-bottom: 5rem;
}
@media (min-width: 576px) {
	.config-list {
		grid-gap: 2rem;
	}
}
.wip {
	opacity: 0.2 !important;
	display: none !important;
}
/* stacked cards; minmax lets wide content scroll inside instead of growing the track */
.detail-panel .config-list {
	grid-template-columns: minmax(0, 1fr);
	margin-bottom: 0;
}
</style>
