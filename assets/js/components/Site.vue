<template>
	<div class="d-flex flex-column site safe-area-inset">
		<div class="container px-4 top-area">
			<div class="d-flex justify-content-between align-items-center mb-2">
				<h1 class="d-block my-0">
					{{ siteTitle || "evcc" }}
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
			<Energyflow v-bind="energyflow" />
		</div>
		<div class="d-flex flex-column justify-content-between content-area">
			<Loadpoints
				class="mt-1 mt-sm-2 flex-grow-1"
				:loadpoints="loadpoints"
				:vehicles="vehicleList"
				:smartCostLimit="smartCostLimit"
				:smartCostType="smartCostType"
				:smartCostActive="smartCostActive"
				:tariffGrid="tariffGrid"
				:tariffCo2="tariffCo2"
				:currency="currency"
			/>
			<Footer v-bind="footer"></Footer>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/arrowup";
import TopNavigation from "./TopNavigation.vue";
import Notifications from "./Notifications.vue";
import Energyflow from "./Energyflow/Energyflow.vue";
import Loadpoints from "./Loadpoints.vue";
import Footer from "./Footer.vue";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";

export default {
	name: "Site",
	components: {
		Loadpoints,
		Energyflow,
		Footer,
		Notifications,
		TopNavigation,
	},
	mixins: [formatter, collector],
	props: {
		loadpoints: Array,

		notifications: Array,
		offline: Boolean,

		// details
		gridConfigured: Boolean,
		gridPower: Number,
		homePower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		pv: Array,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoc: Number,
		batteryDischargeControl: Boolean,
		batterySmartCostLimit: Number,
		batteryMode: String,
		battery: Array,
		gridCurrents: Array,
		prioritySoc: Number,
		bufferSoc: Number,
		bufferStartSoc: Number,
		siteTitle: String,
		vehicles: Object,

		auth: Object,

		currency: String,
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
		sponsor: String,
		sponsorTokenExpires: Number,
		smartCostLimit: Number,
		smartCostType: String,
		smartCostActive: Boolean,
	},
	computed: {
		energyflow: function () {
			return this.collectProps(Energyflow);
		},
		loadpointTitles: function () {
			return this.loadpoints.map((lp) => lp.title);
		},
		loadpointsCompact: function () {
			return this.loadpoints.map((lp) => {
				const vehicleIcon = this.vehicles?.[lp.vehicleName]?.icon;
				const icon = lp.chargerIcon || vehicleIcon || "car";
				const charging = lp.charging;
				const power = lp.chargePower || 0;
				return { icon, charging, power };
			});
		},
		vehicleList: function () {
			const vehicles = this.vehicles || {};
			return Object.entries(vehicles).map(([name, vehicle]) => ({ name, ...vehicle }));
		},
		topNavigation: function () {
			const vehicleLogins = this.auth ? this.auth.vehicles : {};
			return { vehicleLogins, ...this.collectProps(TopNavigation) };
		},
		showParkingLot: function () {
			// work in progess
			return false;
		},
		footer: function () {
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
				sponsor: this.sponsor,
				savings: {
					statistics: this.statistics,
					co2Configured: this.tariffCo2 !== undefined,
					priceConfigured: this.tariffGrid !== undefined,
					currency: this.currency,
				},
			};
		},
	},
};
</script>
<style scoped>
.site {
	min-height: 100vh;
}
.content-area {
	flex-grow: 1;
	z-index: 1;
}
</style>
