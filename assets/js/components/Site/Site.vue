<template>
	<div class="d-flex flex-column site safe-area-inset">
		<div class="container px-4 top-area">
			<div
				class="d-flex justify-content-between align-items-center my-3 my-md-4"
				data-testid="header"
			>
				<h1 class="d-block my-0">
					<span v-if="!isInitialSetup">
						{{ siteTitle || "evcc" }}
					</span>
				</h1>
				<div class="d-flex">
					<Notifications
						:notifications="notifications"
						:loadpointTitles="loadpointTitles"
						class="me-2"
					/>
					<TopNavigation v-bind="topNavigation" />
				</div>
			</div>
			<Energyflow v-if="loadpoints.length > 0" v-bind="energyflow" />
		</div>
		<div class="d-flex flex-column justify-content-between content-area">
			<div v-if="fatal" class="flex-grow-1 align-items-center d-flex justify-content-center">
				<h1 class="mb-5 text-gray fs-4">{{ $t("startupError.title") }}</h1>
			</div>
			<div
				v-else-if="isInitialSetup"
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
				:loadpoints="loadpoints"
				:vehicles="vehicleList"
				:smartCostType="smartCostType"
				:tariffGrid="tariffGrid"
				:tariffCo2="tariffCo2"
				:currency="currency"
				:gridConfigured="gridConfigured"
				:pvConfigured="pvConfigured"
				:batteryConfigured="batteryConfigured"
				:batterySoc="batterySoc"
				:forecast="forecast"
				:selectedIndex="selectedLoadpointIndex"
				@index-changed="selectedLoadpointChanged"
			/>
			<Footer v-bind="footer"></Footer>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/arrowup";
import Navigation from "../Top/Navigation.vue";
import Notifications from "../Top/Notifications.vue";
import Energyflow from "../Energyflow/Energyflow.vue";
import Loadpoints from "../Loadpoints/Loadpoints.vue";
import Footer from "../Footer/Footer.vue";
import formatter from "@/mixins/formatter";
import collector from "@/mixins/collector";
import WelcomeIcons from "./WelcomeIcons.vue";
import { defineComponent, type PropType } from "vue";
import type {
	Auth,
	Battery,
	CURRENCY,
	Forecast,
	LoadpointCompact,
	Notification,
	Sponsor,
} from "@/types/evcc";
import type { Grid } from "./types";

export default defineComponent({
	name: "Site",
	components: {
		Loadpoints,
		Energyflow,
		Footer,
		Notifications,
		TopNavigation: Navigation,
		WelcomeIcons,
	},
	mixins: [formatter, collector],
	props: {
		loadpoints: { type: Array as PropType<LoadpointCompact[]>, default: () => [] },
		selectedLoadpointIndex: Number,

		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
		offline: Boolean,

		// details
		gridConfigured: Boolean,
		grid: Object as PropType<Grid>,
		homePower: Number,
		pvPower: Number,
		pv: { type: Array, default: () => [] },
		batteryPower: Number,
		batterySoc: Number,
		batteryDischargeControl: Boolean,
		batteryGridChargeLimit: { type: Number, default: null },
		batteryGridChargeActive: Boolean,
		batteryMode: String,
		battery: { type: Array as PropType<Battery[]>, default: () => [] },
		gridCurrents: Array,
		prioritySoc: Number,
		bufferSoc: Number,
		bufferStartSoc: Number,
		siteTitle: String,
		vehicles: Object,

		auth: { type: Object as PropType<Auth>, default: () => ({ vehicles: {} }) },

		currency: { type: String as PropType<CURRENCY>, required: true },
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
		smartCostType: String,
		fatal: Object,
		forecast: Object as PropType<Forecast>,
	},
	computed: {
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
		loadpointTitles() {
			return this.loadpoints.map((lp) => lp.title);
		},
		loadpointsCompact() {
			return this.loadpoints.map((lp, index) => {
				const vehicleIcon = this.vehicles?.[lp.vehicleName]?.icon;
				const icon = lp.chargerIcon || vehicleIcon || "car";
				const title =
					this.vehicleTitle(lp.vehicleName) ||
					lp.title ||
					this.$t("main.loadpoint.fallbackName");
				const charging = lp.charging;
				const soc = lp.vehicleSoc;
				const power = lp.chargePower || 0;
				const heating = lp.chargerFeatureHeating;
				return { icon, title, charging, power, soc, heating, index };
			});
		},
		vehicleList() {
			const vehicles = this.vehicles || {};
			return Object.entries(vehicles).map(([name, vehicle]) => ({ name, ...vehicle }));
		},
		topNavigation() {
			return { vehicleLogins: this.auth.vehicles, ...this.collectProps(Navigation) };
		},
		showParkingLot() {
			// work in progess
			return false;
		},
		isInitialSetup() {
			return this.loadpoints.length === 0;
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
				},
			};
		},
	},
	methods: {
		selectedLoadpointChanged(index: number) {
			this.$router.push({ query: { lp: index + 1 } });
		},
		vehicleTitle(vehicleName: string) {
			return this.vehicles?.[vehicleName]?.title;
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
