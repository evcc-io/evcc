<template>
	<div class="d-flex flex-column site safe-area-inset">
		<div class="container px-4 top-area">
			<TopHeader :title="headerTitle" :notifications="notifications" />
			<div
				v-if="batteryConfigWarnings.length && !batteryConfigWarningsDismissed"
				class="alert alert-warning mt-2 mb-3"
				role="alert"
				data-testid="battery-config-warnings"
			>
				<div class="d-flex justify-content-between align-items-start gap-3 mb-1">
					<strong>{{ $t("main.batteryConfigWarnings.title") }}</strong>
					<button
						type="button"
						class="btn-close flex-shrink-0"
						:aria-label="$t('config.general.close')"
						@click="dismissBatteryConfigWarnings"
					></button>
				</div>
				<ul class="mb-2 ps-3">
					<li v-for="warning in batteryConfigWarnings" :key="warningKey(warning)">
						{{ warningMessage(warning) }}
					</li>
				</ul>
				<router-link to="/config" class="alert-link">
					{{ $t("main.batteryConfigWarnings.review") }}
				</router-link>
			</div>
			<HemsWarning :status="hems?.status" />
			<Energyflow v-if="!setupRequired && !hasFatalError" v-bind="energyflow" />
		</div>
		<div class="d-flex flex-column justify-content-between content-area">
			<div
				v-if="hasFatalError"
				class="flex-grow-1 align-items-center d-flex justify-content-center"
			>
				<div class="d-flex flex-column align-items-center mb-5 gap-4 mx-4 text-center">
					<h1 class="text-gray fs-4 my-0">{{ $t("startupError.title") }}</h1>
					<p v-for="fatalText in fatalTexts" :key="fatalText" class="text-break my-0">
						{{ fatalText }}
					</p>
					<router-link class="btn btn-secondary" to="/config">
						{{ $t("startupError.editConfiguration") }}
					</router-link>
				</div>
			</div>
			<div v-else-if="setupRequired" class="flex-grow-1 d-flex p-3" data-testid="setup-empty">
				<div class="empty-box d-flex flex-column p-5">
					<h2 class="fs-4 mb-2">{{ $t("main.setup.title") }}</h2>
					<p class="text-muted mb-4">{{ $t("main.setup.description") }}</p>
					<ul class="list-unstyled mb-4">
						<li
							v-for="{ item, icon } in setupItems"
							:key="item"
							class="d-flex align-items-center gap-2 mb-2"
						>
							<component
								:is="icon"
								size="s"
								class="icon-row text-muted flex-shrink-0"
							/>
							<span>{{ $t(`onboarding.item.${item}`) }}</span>
						</li>
					</ul>
					<router-link to="/config" class="btn btn-outline-primary">
						{{ $t("main.setup.start") }}
					</router-link>
				</div>
			</div>
			<Loadpoints
				v-else
				:key="`loadpoints-${orderedVisibleLoadpoints.length}`"
				class="mt-1 mt-sm-2 flex-grow-1"
				:loadpoints="orderedVisibleLoadpoints"
				:vehicles="vehicleList"
				:smartCostType="smartCostType"
				:smartCostAvailable="smartCostAvailable"
				:smartFeedInPriorityAvailable="smartFeedInPriorityAvailable"
				:tariffGrid="tariffGrid"
				:tariffCo2="tariffCo2"
				:tariffFeedIn="tariffFeedIn"
				:currency="currency"
				:gridConfigured="gridConfigured"
				:gridConnection="gridConnection"
				:voltage="voltage"
				:pvConfigured="pvConfigured"
				:batteryConfigured="batteryConfigured"
				:batterySoc="batterySoc"
				:batteryMode="batteryMode"
				:forecast="forecast"
				:selectedId="selectedLoadpointId"
				@id-changed="selectedLoadpointChanged"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/arrowup";
import TopHeader from "../Top/Header.vue";
import Energyflow from "../Energyflow/Energyflow.vue";
import HemsWarning from "../HemsWarning.vue";
import Loadpoints from "../Loadpoints/Loadpoints.vue";
import formatter from "@/mixins/formatter";
import collector from "@/mixins/collector.ts";
import { ONBOARDING_ICONS } from "../Config/Onboarding/icons";
import { defineComponent, reactive, type PropType } from "vue";
import type {
	AuthProviders,
	Battery,
	Meter,
	CURRENCY,
	GRID_CONNECTION,
	UiForecast,
	Notification,
	ConfigStatus,
	HemsConfig,
	HemsStatus,
	SMART_COST_TYPE,
	FatalError,
	EvOpt,
	BATTERY_MODE,
	Vehicle,
	BatteryConfigWarning,
} from "@/types/evcc";
import vehicleList from "@/utils/vehicleList";
import store from "@/store";
import type { Grid } from "./types";

const batteryConfigWarningState = reactive({ dismissed: false });
const BATTERY_CONFIG_WARNING_KEYS: Record<BatteryConfigWarning["reason"], string> = {
	invalidMaster: "main.batteryConfigWarnings.invalidMaster",
	masterIsSlave: "main.batteryConfigWarnings.masterIsSlave",
	unsupportedSlave: "main.batteryConfigWarnings.unsupportedSlave",
};

export default defineComponent({
	name: "Site",
	components: {
		Loadpoints,
		Energyflow,
		HemsWarning,
		TopHeader,
	},
	mixins: [formatter, collector],
	props: {
		selectedLoadpointId: String,

		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
		offline: Boolean,
		setupRequired: Boolean,

		// details
		gridConfigured: Boolean,
		gridConnection: String as PropType<GRID_CONNECTION>,
		voltage: Number,
		grid: Object as PropType<Grid>,
		homePower: Number,
		pvPower: Number,
		pv: { type: Array as PropType<Meter[]>, default: () => [] },
		aux: { type: Array as PropType<Meter[]>, default: () => [] },
		ext: { type: Array as PropType<Meter[]>, default: () => [] },
		consumers: { type: Array as PropType<Meter[]>, default: () => [] },
		batteryDischargeControl: Boolean,
		solarAdjusted: Boolean,
		batteryGridChargeLimit: { type: [Number, null] as PropType<number | null>, default: null },
		batteryGridChargeActive: Boolean,
		batteryMode: String as PropType<BATTERY_MODE>,
		battery: { type: Object as PropType<Battery> },
		gridCurrents: Array,
		prioritySoc: Number,
		bufferSoc: Number,
		bufferStartSoc: Number,
		siteTitle: String,
		vehicles: Object as PropType<Record<string, Vehicle>>,
		authProviders: { type: Object as PropType<AuthProviders>, default: () => ({}) },
		currency: { type: String as PropType<CURRENCY> },
		tariffFeedIn: Number,
		tariffGrid: Number,
		tariffCo2: Number,
		tariffPriceHome: Number,
		tariffCo2Home: Number,
		tariffPriceLoadpoints: Number,
		tariffCo2Loadpoints: Number,

		smartCostType: String as PropType<SMART_COST_TYPE>,
		smartCostAvailable: Boolean,
		smartFeedInPriorityAvailable: Boolean,
		fatal: { type: Array as PropType<FatalError[]>, default: () => [] },
		batteryConfigWarnings: {
			type: Array as PropType<BatteryConfigWarning[]>,
			default: () => [],
		},
		forecast: Object as PropType<UiForecast>,
		hems: Object as PropType<ConfigStatus<HemsConfig, HemsStatus>>,
		evopt: { type: Object as PropType<EvOpt> },
	},
	computed: {
		setupItems() {
			return (["pv", "battery", "grid", "loadpoint", "heater"] as const).map((item) => ({
				item,
				icon: ONBOARDING_ICONS[item],
			}));
		},
		headerTitle() {
			return this.setupRequired ? "" : this.siteTitle || "evcc";
		},
		loadpoints() {
			return store.uiLoadpoints.value || [];
		},
		enabledLoadpoints() {
			return this.loadpoints.filter((lp) => !lp.disabled);
		},
		orderedVisibleLoadpoints() {
			return this.enabledLoadpoints.filter((lp) => lp.visible);
		},
		batterySoc() {
			return this.battery?.soc;
		},
		batteryConfigured() {
			return (this.battery?.devices?.length ?? 0) > 0;
		},
		batteryConfigWarningsDismissed() {
			return batteryConfigWarningState.dismissed;
		},
		pvConfigured() {
			return this.pv?.length > 0;
		},
		gridPower() {
			return this.grid?.power || 0;
		},
		experimental() {
			return store.state?.experimental;
		},
		energyflow() {
			return { ...this.collectProps(Energyflow), loadpoints: this.enabledLoadpoints };
		},
		vehicleList() {
			return vehicleList(this.vehicles);
		},
		showParkingLot() {
			// work in progess
			return false;
		},
		hasFatalError() {
			return this.fatal.length > 0;
		},
		fatalTexts() {
			return this.fatal.map(({ error, class: errorClass }) =>
				errorClass ? `${errorClass}: ${error}` : error
			);
		},
	},
	methods: {
		dismissBatteryConfigWarnings() {
			batteryConfigWarningState.dismissed = true;
		},
		warningKey(warning: BatteryConfigWarning) {
			return `${warning.device}:${warning.master}:${warning.reason}`;
		},
		warningMessage(warning: BatteryConfigWarning) {
			return this.$t(BATTERY_CONFIG_WARNING_KEYS[warning.reason], {
				device: warning.device,
				master: warning.master,
			});
		},
		selectedLoadpointChanged(id: string | undefined) {
			this.$router.push({ query: { lp: id } });
		},
	},
});
</script>
<style scoped>
.site {
	min-height: 100vh;
	min-height: 100dvh;
}
.content-area {
	flex-grow: 1;
	z-index: 1;
}
</style>
