<template>
	<div class="loadpoint d-flex flex-column pt-4 pb-2 px-3 px-sm-4 mx-2 mx-sm-0">
		<div class="d-flex justify-content-between align-items-center mb-3 text-truncate">
			<h3 class="me-2 mb-0 text-truncate d-flex">
				<VehicleIcon :name="icon" class="me-2 flex-shrink-0" />
				<div class="text-truncate">
					{{ title || $t("main.loadpoint.fallbackName") }}
				</div>
			</h3>
			<LoadpointSettingsButton v-if="settingsButtonVisible" :id="id" />
		</div>
		<LoadpointSettingsModal
			v-bind="settingsModal"
			@maxcurrent-updated="setMaxCurrent"
			@mincurrent-updated="setMinCurrent"
			@phasesconfigured-updated="setPhasesConfigured"
			@minsoc-updated="setMinSoc"
		/>
		<MiniMode class="mb-3" :mode="mode" />

		<div class="details d-flex align-items-start pb-2">
			<div>
				<div class="d-flex align-items-center">
					<LabelAndValue
						:label="$t('main.loadpoint.power')"
						:value="chargePower"
						:valueFmt="fmtPower"
						class="mb-1"
						align="start"
					/>
				</div>
				<Phases v-bind="phasesProps" class="opacity-transiton opacity-100" />
			</div>
			<TargetCharge
				v-if="chargerIcon === 'heater'"
				class="flex-grow-1 text-center target-charge"
				@target-time-updated="setTargetTime"
				@target-time-removed="removeTargetTime"
			/>
			<LabelAndValue
				:label="info.label"
				:value="info.value"
				:extraValue="info.extra"
				align="end"
			/>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/adjust";
import api from "../api";
import Vehicle from "./Vehicle.vue";
import Phases from "./Phases.vue";
import LabelAndValue from "./LabelAndValue.vue";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";
import LoadpointSettingsButton from "./LoadpointSettingsButton.vue";
import LoadpointSettingsModal from "./LoadpointSettingsModal.vue";
import VehicleIcon from "./VehicleIcon";
import TargetCharge from "./TargetCharge.vue";
import MiniMode from "./MiniMode.vue";

export default {
	name: "MiniLoadpoint",
	components: {
		LabelAndValue,
		LoadpointSettingsButton,
		LoadpointSettingsModal,
		VehicleIcon,
		TargetCharge,
		MiniMode,
		Phases,
	},
	mixins: [formatter, collector],
	props: {
		id: Number,
		single: Boolean,

		// main
		title: String,
		mode: String,
		targetSoc: Number,
		targetEnergy: Number,
		remoteDisabled: Boolean,
		remoteDisabledSource: String,
		chargeDuration: Number,
		charging: Boolean,

		// charger
		chargerFeatureIntegratedDevice: Boolean,
		chargerIcon: String,

		// vehicle
		connected: Boolean,
		// charging: Boolean,
		enabled: Boolean,
		vehicleDetectionActive: Boolean,
		vehiclePresent: Boolean,
		vehicleRange: Number,
		vehicleSoc: Number,
		vehicleTitle: String,
		vehicleIcon: String,
		vehicleTargetSoc: Number,
		vehicleCapacity: Number,
		vehicleFeatureOffline: Boolean,
		vehicles: Array,
		minSoc: Number,
		planActive: Boolean,
		planProjectedStart: String,
		targetTime: String,
		vehicleProviderLoggedIn: Boolean,
		vehicleProviderLoginPath: String,
		vehicleProviderLogoutPath: String,

		// details
		chargePower: Number,
		chargedEnergy: Number,
		// chargeDuration: Number,
		climaterActive: Boolean,
		chargeRemainingDuration: Number,

		// other information
		phases: Number,
		phasesConfigured: Number,
		minCurrent: Number,
		maxCurrent: Number,
		phasesActive: Number,
		chargeCurrent: Number,
		connectedDuration: Number,
		chargeCurrents: Array,
		chargeRemainingEnergy: Number,
		phaseAction: String,
		phaseRemaining: Number,
		pvRemaining: Number,
		pvAction: String,
		guardRemaining: Number,
		guardAction: String,
	},
	data() {
		return {
			tickerHandler: null,
			phaseRemainingInterpolated: this.phaseRemaining,
			pvRemainingInterpolated: this.pvRemaining,
			guardRemainingInterpolated: this.guardRemaining,
			chargeDurationInterpolated: this.chargeDuration,
			chargeRemainingDurationInterpolated: this.chargeRemainingDuration,
		};
	},
	computed: {
		icon: function () {
			return this.chargerIcon || this.vehicleIcon;
		},
		integratedDevice: function () {
			return this.chargerFeatureIntegratedDevice;
		},
		phasesProps: function () {
			return this.collectProps(Phases);
		},
		settingsModal: function () {
			return this.collectProps(LoadpointSettingsModal);
		},
		settingsButtonVisible: function () {
			return this.$hiddenFeatures() || [0, 1, 3].includes(this.phasesConfigured);
		},
		vehicle: function () {
			return this.collectProps(Vehicle);
		},
		showChargingIndicator: function () {
			return this.charging && this.chargePower > 0;
		},
		socBasedCharging: function () {
			return !this.vehicleFeatureOffline && this.vehiclePresent;
		},
		info: function () {
			if (this.chargerIcon === "cooler") {
				return { label: "Ø Heute", value: "12 ct/kWh" };
			}
			if (this.chargerIcon === "heater") {
				return { label: "Temp.", value: "54 °C" };
			}
			if (this.chargerIcon === "waterheater") {
				return { label: "Σ 30 Tage", value: this.fmtEnergy(this.chargedEnergy) };
			}
			return { label: "Σ Heute", value: this.fmtEnergy(this.chargedEnergy) };
		},
	},
	watch: {
		phaseRemaining() {
			this.phaseRemainingInterpolated = this.phaseRemaining;
		},
		pvRemaining() {
			this.pvRemainingInterpolated = this.pvRemaining;
		},
		guardRemaining() {
			this.guardRemainingInterpolated = this.guardRemaining;
		},
		chargeDuration() {
			this.chargeDurationInterpolated = this.chargeDuration;
		},
		chargeRemainingDuration() {
			this.chargeRemainingDurationInterpolated = this.chargeRemainingDuration;
		},
	},
	mounted() {
		this.tickerHandler = setInterval(this.tick, 1000);
	},
	unmounted() {
		clearInterval(this.tickerHandler);
	},
	methods: {
		tick() {
			if (this.phaseRemainingInterpolated > 0) {
				this.phaseRemainingInterpolated--;
			}
			if (this.pvRemainingInterpolated > 0) {
				this.pvRemainingInterpolated--;
			}
			if (this.guardRemainingInterpolated > 0) {
				this.guardRemainingInterpolated--;
			}
			if (this.chargeDurationInterpolated > 0 && this.charging) {
				this.chargeDurationInterpolated++;
			}
			if (this.chargeRemainingDurationInterpolated > 0 && this.charging) {
				this.chargeRemainingDurationInterpolated--;
			}
		},
		apiPath: function (func) {
			return "loadpoints/" + this.id + "/" + func;
		},
		setTargetMode: function (mode) {
			api.post(this.apiPath("mode") + "/" + mode);
		},
		setTargetSoc: function (soc) {
			api.post(this.apiPath("target/soc") + "/" + soc);
		},
		setTargetEnergy: function (kWh) {
			api.post(this.apiPath("target/energy") + "/" + kWh);
		},
		setMaxCurrent: function (maxCurrent) {
			api.post(this.apiPath("maxcurrent") + "/" + maxCurrent);
		},
		setMinCurrent: function (minCurrent) {
			api.post(this.apiPath("mincurrent") + "/" + minCurrent);
		},
		setPhasesConfigured: function (phases) {
			api.post(this.apiPath("phases") + "/" + phases);
		},
		setMinSoc: function (soc) {
			api.post(this.apiPath("minsoc") + "/" + soc);
		},
		setTargetTime: function (date) {
			api.post(`${this.apiPath("target/time")}/${date.toISOString()}`);
		},
		removeTargetTime: function () {
			api.delete(this.apiPath("target/time"));
		},
		changeVehicle(index) {
			api.post(this.apiPath("vehicle") + `/${index}`);
		},
		removeVehicle() {
			api.delete(this.apiPath("vehicle"));
		},
		fmtPower(value) {
			const inKw = value == 0 || value >= 1000;
			return this.fmtKw(value, inKw);
		},
		fmtEnergy(value) {
			const inKw = value == 0 || value >= 1000;
			return this.fmtKWh(value, inKw);
		},
	},
};
</script>

<style scoped>
.loadpoint {
	border-radius: 2rem;
	color: var(--evcc-default-text);
	background: var(--evcc-box);
}

.details > div {
	flex-grow: 1;
	flex-basis: 0;
}

.details > div:nth-child(2) {
	text-align: center;
}
.details > div:nth-child(3) {
	text-align: right;
}
.opacity-transiton {
	transition: opacity var(--evcc-transition-slow) ease-in;
}
.divider {
	border: none;
	border-bottom-width: 1px;
	border-bottom-style: solid;
	border-bottom-color: var(--evcc-gray);
	background: none;
	opacity: 0.5;
	margin: 0 -1rem;
}
/* breakpoint sm */
@media (min-width: 576px) {
	.divider {
		margin: 0 -1.5rem;
	}
}
</style>
