<template>
	<div class="loadpoint pt-4 pb-2 px-3 px-sm-4 mx-2 mx-sm-0">
		<div class="d-block d-sm-flex justify-content-between align-items-center mb-3">
			<div class="d-flex justify-content-between align-items-center mb-3">
				<h3 class="me-2 mb-0 text-truncate">
					{{ title || $t("main.loadpoint.fallbackName") }}
				</h3>
				<LoadpointSettingsButton :id="id" class="d-block d-sm-none" />
			</div>
			<div class="mb-3 d-flex align-items-center">
				<Mode class="flex-grow-1" :mode="mode" @updated="setTargetMode" />
				<LoadpointSettingsButton :id="id" class="d-none d-sm-block ms-2" />
			</div>
		</div>
		<LoadpointSettingsModal
			v-bind="settingsModal"
			@maxcurrent-updated="setMaxCurrent"
			@mincurrent-updated="setMinCurrent"
			@phasesconfigured-updated="setPhasesConfigured"
			@minsoc-updated="setMinSoC"
		/>

		<div
			v-if="remoteDisabled"
			class="alert alert-warning my-4 py-2"
			:class="`${remoteDisabled === 'hard' ? 'alert-danger' : 'alert-warning'}`"
			role="alert"
		>
			{{
				$t(
					remoteDisabled === "hard"
						? "main.loadpoint.remoteDisabledHard"
						: "main.loadpoint.remoteDisabledSoft",
					{ source: remoteDisabledSource }
				)
			}}
		</div>

		<div class="details d-flex align-items-start mb-3">
			<div>
				<div class="d-flex align-items-center">
					<LabelAndValue
						:label="$t('main.loadpoint.power')"
						:value="chargePower"
						:valueFmt="fmtkWUnit"
						class="mb-2"
						align="start"
					/>
					<shopicon-regular-lightning
						class="text-evcc opacity-transiton"
						:class="`opacity-${showChargingIndicator ? '100' : '0'}`"
						size="m"
					></shopicon-regular-lightning>
				</div>
				<Phases
					v-bind="phasesProps"
					class="opacity-transiton"
					:class="`opacity-${showChargingIndicator ? '100' : '0'}`"
				/>
			</div>
			<LabelAndValue
				:label="$t('main.loadpoint.charged')"
				:value="fmtKWh(chargedEnergy)"
				align="center"
			/>
			<LabelAndValue
				v-if="chargeRemainingDurationInterpolated"
				:label="$t('main.loadpoint.remaining')"
				:value="`
					${fmtShortDuration(chargeRemainingDurationInterpolated)}
					${fmtShortDurationUnit(chargeRemainingDurationInterpolated, true)}`"
				align="end"
			/>
			<LabelAndValue
				v-else
				:label="$t('main.loadpoint.duration')"
				:value="`
					${fmtShortDuration(chargeDurationInterpolated)}
					${fmtShortDurationUnit(chargeDurationInterpolated)}`"
				align="end"
			/>
		</div>
		<hr class="divider" />
		<Vehicle
			v-bind="vehicle"
			@target-soc-updated="setTargetSoC"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
			@change-vehicle="changeVehicle"
			@remove-vehicle="removeVehicle"
		/>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/adjust";
import api from "../api";
import Mode from "./Mode.vue";
import Vehicle from "./Vehicle.vue";
import Phases from "./Phases.vue";
import LabelAndValue from "./LabelAndValue.vue";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";
import LoadpointSettingsButton from "./LoadpointSettingsButton.vue";
import LoadpointSettingsModal from "./LoadpointSettingsModal.vue";

export default {
	name: "Loadpoint",
	components: {
		Mode,
		Vehicle,
		Phases,
		LabelAndValue,
		LoadpointSettingsButton,
		LoadpointSettingsModal,
	},
	mixins: [formatter, collector],
	props: {
		id: Number,
		single: Boolean,

		// main
		title: String,
		mode: String,
		targetSoC: Number,
		remoteDisabled: Boolean,
		remoteDisabledSource: String,
		chargeDuration: Number,
		charging: Boolean,

		// vehicle
		connected: Boolean,
		// charging: Boolean,
		enabled: Boolean,
		vehicleTitle: String,
		vehicleSoC: Number,
		vehiclePresent: Boolean,
		vehicleRange: Number,
		vehicles: Array,
		minSoC: Number,
		targetTime: String,
		targetTimeActive: Boolean,
		targetTimeProjectedStart: String,
		vehicleProviderLoggedIn: Boolean,
		vehicleProviderLoginPath: String,
		vehicleProviderLogoutPath: String,

		// details
		chargePower: Number,
		chargedEnergy: Number,
		// chargeDuration: Number,
		climater: String,
		chargeRemainingDuration: Number,

		// other information
		phases: Number,
		phasesConfigured: Number,
		minCurrent: Number,
		maxCurrent: Number,
		phasesActive: Number,
		chargeCurrent: Number,
		vehicleCapacity: Number,
		connectedDuration: Number,
		chargeCurrents: Array,
		chargeConfigured: Boolean,
		chargeRemainingEnergy: Number,
		phaseAction: String,
		phaseRemaining: Number,
		pvRemaining: Number,
		pvAction: String,
	},
	data() {
		return {
			tickerHandler: null,
			phaseRemainingInterpolated: this.phaseRemaining,
			pvRemainingInterpolated: this.pvRemaining,
			chargeDurationInterpolated: this.chargeDuration,
			chargeRemainingDurationInterpolated: this.chargeRemainingDuration,
		};
	},
	computed: {
		phasesProps: function () {
			return this.collectProps(Phases);
		},
		settingsModal: function () {
			return this.collectProps(LoadpointSettingsModal);
		},
		vehicle: function () {
			return this.collectProps(Vehicle);
		},
		showChargingIndicator: function () {
			return this.charging && this.chargePower > 0;
		},
	},
	watch: {
		phaseRemaining() {
			this.phaseRemainingInterpolated = this.phaseRemaining;
		},
		pvRemaining() {
			this.pvRemainingInterpolated = this.pvRemaining;
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
		setTargetSoC: function (soc) {
			api.post(this.apiPath("targetsoc") + "/" + soc);
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
		setMinSoC: function (soc) {
			api.post(this.apiPath("minsoc") + "/" + soc);
		},
		setTargetTime: function (date) {
			api.post(`${this.apiPath("targetcharge")}/${this.targetSoC}/${date.toISOString()}`);
		},
		removeTargetTime: function () {
			api.delete(this.apiPath("targetcharge"));
		},
		changeVehicle(index) {
			api.post(this.apiPath("vehicle") + `/${index}`);
		},
		removeVehicle() {
			api.delete(this.apiPath("vehicle"));
		},
		fmtkWUnit(value) {
			const inKw = value == 0 || value >= 1000;
			return this.fmtKw(value, inKw);
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
@media (--sm-and-up) {
	.divider {
		margin: 0 -1.5rem;
	}
}
</style>
