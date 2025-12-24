<template>
	<div class="d-flex flex-column site safe-area-inset">
		<div class="container px-4 top-area">
			<div
				class="d-flex justify-content-between align-items-center my-3 my-md-4"
				data-testid="header"
			>
				<h1 class="d-block my-0">
					<span v-if="!setupRequired">
						{{ siteTitle || "evcc" }}
					</span>
				</h1>
				<TopNavigationArea :notifications="notifications" />
			</div>
			<HemsWarning :circuits="circuits" />
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
			<div
				v-else-if="setupRequired"
				class="flex-grow-1 d-flex align-items-center justify-content-center p-3"
			>
				<div
					class="welcome d-flex align-items-center flex-column justify-content-center text-center"
				>
					<h1 class="mb-0 fs-4 d-flex align-items-center gap-2">
						{{ $t("main.welcome") }}
					</h1>
					<WelcomeIcons class="welcome-icons" />
					<router-link
						class="btn btn-lg btn-outline-primary configure-button"
						to="/config"
					>
						{{ $t("main.startConfiguration") }}
					</router-link>
				</div>
			</div>
			<Loadpoints
				v-else-if="loadpoints.length > 0"
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
				:pvConfigured="pvConfigured"
				:batteryConfigured="batteryConfigured"
				:batterySoc="batterySoc"
				:forecast="forecast"
				:selectedId="selectedLoadpointId"
				@id-changed="selectedLoadpointChanged"
			/>
			<Footer v-bind="footer"></Footer>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/arrowup";
import TopNavigationArea from "../Top/TopNavigationArea.vue";
import Energyflow from "../Energyflow/Energyflow.vue";
import HemsWarning from "../HemsWarning.vue";
import Loadpoints from "../Loadpoints/Loadpoints.vue";
import Footer from "../Footer/Footer.vue";
import formatter from "@/mixins/formatter";
import collector from "@/mixins/collector.ts";
import WelcomeIcons from "./WelcomeIcons.vue";
import { defineComponent, type PropType } from "vue";
import type {
	AuthProviders,
	BatteryMeter,
	Meter,
	CURRENCY,
	Forecast,
	Notification,
	Circuit,
	SMART_COST_TYPE,
	Sponsor,
	FatalError,
	EvOpt,
} from "@/types/evcc";
import store from "@/store";
import type { Grid } from "./types";

export default defineComponent({
	name: "Site",
	components: {
		Loadpoints,
		Energyflow,
		Footer,
		HemsWarning,
		TopNavigationArea,
		WelcomeIcons,
	},
	mixins: [formatter, collector],
	props: {
		selectedLoadpointId: String,

		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
		offline: Boolean,
		setupRequired: Boolean,

		// details
		gridConfigured: Boolean,
		grid: Object as PropType<Grid>,
		homePower: Number,
		pvPower: Number,
		pv: { type: Array as PropType<Meter[]>, default: () => [] },
		aux: { type: Array as PropType<Meter[]>, default: () => [] },
		ext: { type: Array as PropType<Meter[]>, default: () => [] },
		batteryPower: Number,
		batterySoc: Number,
		batteryDischargeControl: Boolean,
		batteryGridChargeLimit: { type: Number, default: null },
		batteryGridChargeActive: Boolean,
		batteryMode: String,
		battery: { type: Array as PropType<BatteryMeter[]>, default: () => [] },
		gridCurrents: Array,
		prioritySoc: Number,
		bufferSoc: Number,
		bufferStartSoc: Number,
		siteTitle: String,
		vehicles: Object,
		authProviders: { type: Object as PropType<AuthProviders>, default: () => ({}) },
		currency: { type: String as PropType<CURRENCY> },
		statistics: Object,
		tariffFeedIn: Number,
		tariffGrid: Number,
		tariffCo2: Number,
		tariffPriceHome: Number,
		tariffCo2Home: Number,
		tariffPriceLoadpoints: Number,
		tariffCo2Loadpoints: Number,

		availableVersion: String,
		releaseNotes: String,
		hasUpdater: Boolean,
		uploadMessage: String,
		uploadProgress: Number,
		sponsor: { type: Object as PropType<Sponsor>, default: () => ({}) },
		smartCostType: String as PropType<SMART_COST_TYPE>,
		smartCostAvailable: Boolean,
		smartFeedInPriorityAvailable: Boolean,
		fatal: { type: Array as PropType<FatalError[]>, default: () => [] },
		forecast: Object as PropType<Forecast>,
		circuits: Object as PropType<Record<string, Circuit>>,
		telemetry: Boolean,
		evopt: { type: Object as PropType<EvOpt> },
	},
	computed: {
		loadpoints() {
			return store.uiLoadpoints.value || [];
		},
		orderedVisibleLoadpoints() {
			return this.loadpoints.filter((lp) => lp.visible);
		},
		batteryConfigured() {
			return this.battery?.length > 0;
		},
		pvConfigured() {
			return this.pv?.length > 0;
		},
		gridPower() {
			return this.grid?.power || 0;
		},
		energyflow() {
			return this.collectProps(Energyflow);
		},
		vehicleList() {
			const vehicles = this.vehicles || {};
			return Object.entries(vehicles).map(([name, vehicle]) => ({ name, ...vehicle }));
		},
		showParkingLot() {
			// work in progess
			return false;
		},
		footer() {
			return {
				version: {
					installed: window.evcc.version,
					commit: window.evcc.commit,
					available: this.availableVersion,
					releaseNotes: this.releaseNotes,
					hasUpdater: this.hasUpdater,
					uploadMessage: this.uploadMessage,
					uploadProgress: this.uploadProgress,
				},
				savings: {
					sponsor: this.sponsor,
					statistics: this.statistics,
					co2Configured: this.tariffCo2 !== undefined,
					priceConfigured: this.tariffGrid !== undefined,
					currency: this.currency,
					telemetry: this.telemetry,
				},
			};
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
.configure-button:not(:active):not(:hover),
.welcome-icons {
	animation: colorTransition 10s infinite alternate;
	animation-timing-function: ease-in-out;
}

@keyframes colorTransition {
	0% {
		color: var(--evcc-accent1);
		border-color: var(--evcc-accent1);
	}
	50% {
		color: var(--evcc-accent2);
		border-color: var(--evcc-accent2);
	}
	100% {
		color: var(--evcc-accent3);
		border-color: var(--evcc-accent3);
	}
}
.welcome {
	background-color: var(--evcc-box);
	padding: 4rem;
	border-radius: 2rem;
}
</style>
