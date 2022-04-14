<template>
	<div class="d-flex flex-column top-area">
		<div class="container px-4">
			<div class="d-flex justify-content-between align-items-center">
				<h1 class="d-block my-4">
					{{ siteTitle || "evcc" }}
				</h1>
				<div class="py-1 d-flex">
					<Notifications :notifications="notifications" class="me-2" />
					<TopNavigation v-bind="topNavigation" />
				</div>
			</div>
			<Energyflow v-bind="energyflow" @toggle-details="toggleDetails" />
		</div>
		<div
			class="d-flex flex-column content-area"
			:style="{
				transform: `translateY(${detailsVisible ? detailsHeight : 3}px)`,
				'padding-bottom': `${detailsVisible ? detailsHeight : 3}px`,
			}"
		>
			<div
				class="toggle-handle py-3 d-flex justify-content-center mb-3"
				@click="toggleDetails"
			>
				<hr class="toggle-handle-icon bg-white m-0 p-0" />
			</div>
			<Loadpoints :loadpoints="loadpoints" />
			<Vehicles v-if="$hiddenFeatures" />
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
import Vehicles from "./Vehicles.vue";
import Footer from "./Footer.vue";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";

export default {
	name: "Site",
	components: { Loadpoints, Energyflow, Footer, Notifications, TopNavigation, Vehicles },
	mixins: [formatter, collector],
	props: {
		loadpoints: Array,

		notifications: Array,

		// details
		gridConfigured: Boolean,
		gridPower: Number,
		homePower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoC: Number,
		gridCurrents: Array,
		prioritySoC: Number,
		siteTitle: String,

		auth: Object,

		// footer
		currency: String,
		savingsAmount: Number,
		savingsEffectivePrice: Number,
		savingsGridCharged: Number,
		savingsSelfConsumptionCharged: Number,
		savingsSelfConsumptionPercent: Number,
		savingsSince: Number,
		savingsTotalCharged: Number,
		tariffFeedIn: Number,
		tariffGrid: Number,
	},
	data: function () {
		return {
			detailsVisible: false,
			detailsHeight: 0,
			availableVersion: null,
			releaseNotes: null,
			hasUpdater: null,
			uploadMessage: null,
			uploadProgress: null,
			sponsor: null,
		};
	},
	computed: {
		energyflow: function () {
			return this.collectProps(Energyflow);
		},
		activeLoadpointsCount: function () {
			return this.loadpoints.filter((lp) => lp.chargePower > 0).length;
		},
		loadpointsPower: function () {
			return this.loadpoints.reduce((sum, lp) => {
				sum += lp.chargePower || 0;
				return sum;
			}, 0);
		},
		topNavigation: function () {
			const vehicleLogins = this.auth ? this.auth.vehicles : {};
			return { vehicleLogins };
		},
		footer: function () {
			return {
				version: {
					installed: window.evcc.version,
					available: this.availableVersion,
					releaseNotes: this.releaseNotes,
					hasUpdater: this.hasUpdater,
					uploadMessage: this.uploadMessage,
					uploadProgress: this.uploadProgress,
				},
				sponsor: this.sponsor,
				savings: {
					since: this.savingsSince,
					totalCharged: this.savingsTotalCharged,
					gridCharged: this.savingsGridCharged,
					selfConsumptionCharged: this.savingsSelfConsumptionCharged,
					amount: this.savingsAmount,
					effectivePrice: this.savingsEffectivePrice,
					selfConsumptionPercent: this.savingsSelfConsumptionPercent,
					gridPrice: this.tariffGrid,
					feedInPrice: this.tariffFeedIn,
					currency: this.currency,
				},
			};
		},
	},
	mounted() {
		this.updateDetailHeight();
		window.addEventListener("resize", this.updateDetailHeight);
	},
	unmounted() {
		window.removeEventListener("resize", this.updateDetailHeight);
	},
	methods: {
		updateDetailHeight: function () {
			this.detailsHeight = this.$el.querySelector("[data-collapsible-details]").offsetHeight;
		},
		toggleDetails() {
			this.updateDetailHeight();
			this.detailsVisible = !this.detailsVisible;
		},
	},
};
</script>
<style scoped>
.top-area {
	background: linear-gradient(0deg, var(--bs-gray-dark) 5%, var(--bs-white) 5%);
}
.content-area {
	background-color: var(--bs-gray-dark);
	color: var(--bs-white);
	transform: translateY(0);
	transition-property: transform;
	transition-duration: 0.5s;
	transition-timing-function: cubic-bezier(0.5, 0.5, 0.5, 1.15);
}
.toggle-handle {
	cursor: pointer;
	color: var(--bs-gray-medium);
}
.toggle-handle-icon {
	border: none;
	width: 1.75rem;
	height: 2px;
}
</style>
