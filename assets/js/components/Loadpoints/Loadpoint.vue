<template>
	<div class="loadpoint d-flex flex-column pt-4 pb-2 px-3 px-sm-4 mx-2 mx-sm-0">
		<div class="d-block d-sm-flex justify-content-between align-items-center mb-3">
			<div class="d-flex justify-content-between align-items-center mb-3 text-truncate">
				<h3 class="me-2 mb-0 text-truncate d-flex">
					<VehicleIcon
						v-if="chargerIcon"
						:name="chargerIcon"
						class="me-2 flex-shrink-0"
					/>
					<div class="text-truncate">
						{{ loadpointTitle }}
					</div>
				</h3>
				<LoadpointSettingsButton class="d-block d-sm-none" @click="openSettingsModal" />
			</div>
			<div class="mb-3 d-flex align-items-center">
				<Mode class="flex-grow-1" v-bind="modeProps" @updated="setTargetMode" />
				<LoadpointSettingsButton
					:id="id"
					class="d-none d-sm-block ms-2"
					@click="openSettingsModal"
				/>
			</div>
		</div>
		<LoadpointSettingsModal
			v-bind="settingsModal"
			@maxcurrent-updated="setMaxCurrent"
			@mincurrent-updated="setMinCurrent"
			@phasesconfigured-updated="setPhasesConfigured"
			@batteryboost-updated="setBatteryBoost"
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

		<div class="details d-flex align-items-start mb-2">
			<div>
				<div class="d-flex align-items-center">
					<LabelAndValue
						:label="$t('main.loadpoint.power')"
						:value="chargePower"
						:valueFmt="fmtPower"
						class="mb-2 text-nowrap text-truncate-xs-only"
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
				v-show="socBasedCharging"
				:label="$t('main.loadpoint.charged')"
				:value="chargedEnergy"
				:valueFmt="fmtEnergy"
				align="center"
			/>
			<LoadpointSessionInfo v-bind="sessionInfoProps" />
		</div>
		<hr class="divider" />
		<Vehicle
			class="flex-grow-1 d-flex flex-column justify-content-end"
			v-bind="vehicleProps"
			@limit-soc-updated="setLimitSoc"
			@limit-energy-updated="setLimitEnergy"
			@change-vehicle="changeVehicle"
			@remove-vehicle="removeVehicle"
			@open-loadpoint-settings="openSettingsModal"
		/>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/adjust";
import api from "@/api";
import Mode from "./Mode.vue";
import VehicleComponent from "../Vehicles/Vehicle.vue";
import Phases from "./Phases.vue";
import LabelAndValue from "../Helper/LabelAndValue.vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import collector from "@/mixins/collector.js";
import SettingsButton from "./SettingsButton.vue";
import SettingsModal from "./SettingsModal.vue";
import VehicleIcon from "../VehicleIcon";
import SessionInfo from "./SessionInfo.vue";
import Modal from "bootstrap/js/dist/modal";
import { defineComponent, type PropType } from "vue";
import type {
	CHARGE_MODE,
	PHASES,
	PHASE_ACTION,
	PV_ACTION,
	CHARGER_STATUS_REASON,
	Timeout,
	Vehicle,
	Forecast,
	SMART_COST_TYPE,
} from "@/types/evcc";

export default defineComponent({
	name: "Loadpoint",
	components: {
		Mode,
		Vehicle: VehicleComponent,
		Phases,
		LabelAndValue,
		LoadpointSettingsButton: SettingsButton,
		LoadpointSettingsModal: SettingsModal,
		LoadpointSessionInfo: SessionInfo,
		VehicleIcon,
	},
	mixins: [formatter, collector],
	props: {
		id: String,
		single: Boolean,

		// main
		title: String,
		mode: String as PropType<CHARGE_MODE>,
		effectiveLimitSoc: Number,
		limitEnergy: Number,
		remoteDisabled: String,
		remoteDisabledSource: String,
		chargeDuration: { type: Number, default: 0 },
		charging: Boolean,
		batteryBoost: Boolean,
		batteryConfigured: Boolean,

		// session
		sessionEnergy: Number,
		sessionCo2PerKWh: Number as PropType<number | null>,
		sessionPricePerKWh: Number as PropType<number | null>,
		sessionPrice: Number as PropType<number | null>,
		sessionSolarPercentage: Number,

		// charger
		chargerStatusReason: String as PropType<CHARGER_STATUS_REASON | null>,
		chargerFeatureIntegratedDevice: Boolean,
		chargerFeatureHeating: Boolean,
		chargerIcon: String as PropType<string | null>,

		// vehicle
		connected: Boolean,
		// charging: Boolean,
		enabled: Boolean,
		vehicleDetectionActive: Boolean,
		vehicleRange: Number,
		vehicleSoc: { type: Number, default: 0 },
		vehicleName: String,
		vehicleIcon: String,
		vehicleLimitSoc: Number,
		vehicles: Array as PropType<Vehicle[]>,
		planActive: Boolean,
		planProjectedStart: String as PropType<string | null>,
		planProjectedEnd: String as PropType<string | null>,
		planOverrun: { type: Number, default: 0 },
		planEnergy: Number,
		planPrecondition: Number,
		planTime: String as PropType<string | null>,
		effectivePlanTime: String as PropType<string | null>,
		effectivePlanSoc: Number,
		vehicleProviderLoggedIn: Boolean,
		vehicleProviderLoginPath: String,
		vehicleProviderLogoutPath: String,

		// details
		vehicleClimaterActive: Boolean as PropType<boolean | null>,
		vehicleWelcomeActive: Boolean,
		chargePower: { type: Number, default: 0 },
		chargedEnergy: { type: Number, default: 0 },
		chargeRemainingDuration: { type: Number, default: 0 },

		// other information
		phasesConfigured: Number,
		phasesActive: Number,
		chargerPhases1p3p: Boolean,
		chargerSinglePhase: Boolean,
		minCurrent: Number,
		maxCurrent: Number,
		offeredCurrent: Number,
		connectedDuration: Number,
		chargeCurrents: Array,
		chargeRemainingEnergy: Number,
		phaseAction: String as PropType<PHASE_ACTION>,
		phaseRemaining: { type: Number, default: 0 },
		pvRemaining: { type: Number, default: 0 },
		pvAction: String as PropType<PV_ACTION>,
		smartCostLimit: { type: Number as PropType<number | null>, default: null },
		smartCostType: String as PropType<SMART_COST_TYPE>,
		smartCostAvailable: Boolean,
		smartCostActive: Boolean,
		smartCostNextStart: String as PropType<string | null>,
		smartFeedInPriorityLimit: { type: Number as PropType<number | null>, default: null },
		smartFeedInPriorityAvailable: Boolean,
		smartFeedInPriorityActive: Boolean,
		smartFeedInPriorityNextStart: String as PropType<string | null>,
		tariffGrid: Number,
		tariffFeedIn: Number,
		tariffCo2: Number,
		currency: String,
		multipleLoadpoints: Boolean,
		gridConfigured: Boolean,
		pvConfigured: Boolean,
		forecast: Object as PropType<Forecast>,
	},
	data() {
		return {
			tickerHandler: null as Timeout,
			phaseRemainingInterpolated: this.phaseRemaining,
			pvRemainingInterpolated: this.pvRemaining,
			chargeDurationInterpolated: this.chargeDuration,
			chargeRemainingDurationInterpolated: this.chargeRemainingDuration,
		};
	},
	computed: {
		vehicle() {
			return this.vehicles?.find((v) => v.name === this.vehicleName);
		},
		vehicleTitle() {
			return this.vehicle?.title;
		},
		loadpointTitle() {
			return this.title || this.$t("main.loadpoint.fallbackName");
		},
		integratedDevice() {
			return this.chargerFeatureIntegratedDevice;
		},
		heating() {
			return this.chargerFeatureHeating;
		},
		phasesProps() {
			return this.collectProps(Phases);
		},
		modeProps() {
			return this.collectProps(Mode);
		},
		sessionInfoProps() {
			return this.collectProps(SessionInfo);
		},
		settingsModal() {
			return this.collectProps(SettingsModal);
		},
		vehicleProps() {
			return this.collectProps(VehicleComponent);
		},
		showChargingIndicator() {
			return this.charging && this.chargePower > 0;
		},
		vehicleKnown() {
			return !!this.vehicleName;
		},
		vehicleHasSoc() {
			return this.vehicleKnown && !this.vehicle?.features?.includes("Offline");
		},
		vehicleNotReachable() {
			// online vehicle that was not reachable at startup
			const features = this.vehicle?.features || [];
			return features.includes("Offline") && features.includes("Retryable");
		},
		planTimeUnreachable() {
			// 1 minute tolerance
			return this.planOverrun > 60;
		},
		socBasedCharging() {
			return this.vehicleHasSoc || this.vehicleSoc > 0;
		},
		socBasedPlanning() {
			return this.socBasedCharging && this.vehicle?.capacity && this.vehicle?.capacity > 0;
		},
		pvPossible() {
			return this.pvConfigured || this.gridConfigured;
		},
		batteryBoostAvailable() {
			return this.batteryConfigured && this.$hiddenFeatures();
		},
		batteryBoostActive() {
			return (
				this.batteryBoost &&
				this.charging &&
				this.mode &&
				!["off", "now"].includes(this.mode)
			);
		},
		plannerForecast() {
			return this.forecast?.planner;
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
		if (this.tickerHandler) {
			clearInterval(this.tickerHandler);
		}
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
		apiPath(func: string) {
			return "loadpoints/" + this.id + "/" + func;
		},
		setTargetMode(mode: CHARGE_MODE) {
			api.post(this.apiPath("mode") + "/" + mode);
		},
		setLimitSoc(soc: number) {
			api.post(this.apiPath("limitsoc") + "/" + soc);
		},
		setLimitEnergy(kWh: number) {
			api.post(this.apiPath("limitenergy") + "/" + kWh);
		},
		setMaxCurrent(maxCurrent: number) {
			api.post(this.apiPath("maxcurrent") + "/" + maxCurrent);
		},
		setMinCurrent(minCurrent: number) {
			api.post(this.apiPath("mincurrent") + "/" + minCurrent);
		},
		setPhasesConfigured(phases: PHASES) {
			api.post(this.apiPath("phases") + "/" + phases);
		},
		changeVehicle(name: string) {
			api.post(this.apiPath("vehicle") + `/${name}`);
		},
		removeVehicle() {
			api.delete(this.apiPath("vehicle"));
		},
		setBatteryBoost(batteryBoost: boolean) {
			api.post(this.apiPath("batteryboost") + `/${batteryBoost ? "1" : "0"}`);
		},
		fmtPower(value: number) {
			return this.fmtW(value, POWER_UNIT.AUTO);
		},
		fmtEnergy(value: number) {
			return this.fmtWh(value, POWER_UNIT.AUTO);
		},
		openSettingsModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById(`loadpointSettingsModal_${this.id}`) as HTMLElement
			);
			modal.show();
		},
	},
});
</script>

<style scoped>
.loadpoint {
	border-radius: 2rem;
	color: var(--evcc-default-text);
	background: var(--evcc-box);
}

.details > div {
	flex-grow: 1;
	flex-shrink: 1;
	flex-basis: 0;
	min-width: 0;
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
