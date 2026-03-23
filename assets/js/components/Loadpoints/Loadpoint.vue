<template>
	<Teleport to="body" :disabled="!loadpointViewportMaximized">
		<div
			:class="
				loadpointViewportMaximized ? 'loadpoint-viewport-overlay safe-area-inset' : 'loadpoint-inline-host'
			"
		>
			<div
				class="loadpoint d-flex flex-column pt-4 pb-2 px-3 px-sm-4"
				v-bind="loadpointRootAttrs"
				:class="[
					loadpointViewportMaximized
						? 'loadpoint--viewport-expanded flex-grow-1 mx-0'
						: 'mx-2 mx-sm-0',
				]"
			>
				<!-- sm+: title, expand/collapse, mode, settings in one row -->
				<div class="d-none d-sm-flex align-items-center gap-2 mb-3 flex-wrap loadpoint-header-wide">
					<h3 class="mb-0 text-truncate d-flex min-w-0 loadpoint-header-wide__title">
						<VehicleIcon
							v-if="chargerIcon"
							:name="chargerIcon"
							class="me-2 flex-shrink-0"
						/>
						<div class="text-truncate">
							{{ loadpointTitle }}
						</div>
					</h3>
					<button
						type="button"
						class="btn btn-sm btn-outline-secondary border-0 p-2 evcc-gray flex-shrink-0 loadpoint-expand-btn"
						:aria-label="
							loadpointViewportMaximized
								? $t('main.loadpoint.backToOverview')
								: $t('main.loadpoint.expandViewport')
						"
						:data-testid="
							loadpointViewportMaximized
								? 'loadpoint-viewport-back'
								: 'loadpoint-viewport-expand'
						"
						@click.stop="
							loadpointViewportMaximized ? collapseViewport() : expandViewport()
						"
					>
						<svg
							v-if="loadpointViewportMaximized"
							class="loadpoint-expand-icon"
							width="18"
							height="18"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
							focusable="false"
						>
							<path d="M4 14h6v6" />
							<path d="M20 10h-6V4" />
							<path d="M14 10l7-7" />
							<path d="M3 21l7-7" />
						</svg>
						<svg
							v-else
							class="loadpoint-expand-icon"
							width="18"
							height="18"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
							focusable="false"
						>
							<path d="M8 3H5a2 2 0 0 0-2 2v3" />
							<path d="M21 8V5a2 2 0 0 0-2-2h-3" />
							<path d="M3 16v3a2 2 0 0 0 2 2h3" />
							<path d="M16 21h3a2 2 0 0 0 2-2v-3" />
						</svg>
					</button>
					<Mode
						class="loadpoint-header-wide__mode flex-shrink-0"
						v-bind="modeProps"
						@updated="setTargetMode"
					/>
					<LoadpointSettingsButton
						:id="id"
						:class="expandLoadpointHeader ? 'd-lg-none d-xl-block' : ''"
						class="flex-shrink-0"
						@click="openSettingsModal"
					/>
				</div>
				<!-- xs: stacked -->
				<div class="d-flex d-sm-none flex-column mb-3">
					<div class="d-flex justify-content-between align-items-center gap-2 mb-3">
						<div class="d-flex align-items-center gap-2 min-w-0 flex-grow-1">
							<h3 class="mb-0 text-truncate d-flex min-w-0 flex-grow-1">
								<VehicleIcon
									v-if="chargerIcon"
									:name="chargerIcon"
									class="me-2 flex-shrink-0"
								/>
								<div class="text-truncate">
									{{ loadpointTitle }}
								</div>
							</h3>
							<button
								type="button"
								class="btn btn-sm btn-outline-secondary border-0 p-2 evcc-gray flex-shrink-0 loadpoint-expand-btn"
								:aria-label="
									loadpointViewportMaximized
										? $t('main.loadpoint.backToOverview')
										: $t('main.loadpoint.expandViewport')
								"
								:data-testid="
									loadpointViewportMaximized
										? 'loadpoint-viewport-back-xs'
										: 'loadpoint-viewport-expand-xs'
								"
								@click.stop="
									loadpointViewportMaximized
										? collapseViewport()
										: expandViewport()
								"
							>
								<svg
									v-if="loadpointViewportMaximized"
									class="loadpoint-expand-icon"
									width="18"
									height="18"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
									aria-hidden="true"
									focusable="false"
								>
									<path d="M4 14h6v6" />
									<path d="M20 10h-6V4" />
									<path d="M14 10l7-7" />
									<path d="M3 21l7-7" />
								</svg>
								<svg
									v-else
									class="loadpoint-expand-icon"
									width="18"
									height="18"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
									aria-hidden="true"
									focusable="false"
								>
									<path d="M8 3H5a2 2 0 0 0-2 2v3" />
									<path d="M21 8V5a2 2 0 0 0-2-2h-3" />
									<path d="M3 16v3a2 2 0 0 0 2 2h3" />
									<path d="M16 21h3a2 2 0 0 0 2-2v-3" />
								</svg>
							</button>
						</div>
						<LoadpointSettingsButton
							:class="expandLoadpointHeader ? 'd-lg-block d-xl-none' : ''"
							class="flex-shrink-0"
							@click="openSettingsModal"
						/>
					</div>
					<Mode v-bind="modeProps" @updated="setTargetMode" />
				</div>
				<LoadpointSettingsModal
					:id="id"
					v-bind="settingsModal"
					@maxcurrent-updated="setMaxCurrent"
					@mincurrent-updated="setMinCurrent"
					@phasesconfigured-updated="setPhasesConfigured"
					@batteryboostlimit-updated="setBatteryBoostLimit"
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
					@batteryboost-updated="setBatteryBoost"
				/>
			</div>
		</div>
	</Teleport>
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
import type { PlanStrategy } from "@/components/ChargingPlans/types";

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
	inheritAttrs: false,
	props: {
		id: { type: String, required: true },
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
		batteryBoostLimit: { type: Number, default: 100 },
		batteryConfigured: Boolean,
		batterySoc: Number,

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
		minSocNotReached: Boolean,
		vehicleName: String,
		vehicleIcon: String,
		vehicleLimitSoc: Number,
		vehicles: Array as PropType<Vehicle[]>,
		planActive: Boolean,
		planProjectedStart: String as PropType<string | null>,
		planProjectedEnd: String as PropType<string | null>,
		planOverrun: { type: Number, default: 0 },
		planEnergy: Number,
		planTime: String as PropType<string | null>,
		effectivePlanTime: String as PropType<string | null>,
		effectivePlanSoc: Number,
		effectivePlanStrategy: Object as PropType<PlanStrategy>,
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
		fullWidth: Boolean,
		gridConfigured: Boolean,
		pvConfigured: Boolean,
		forecast: Object as PropType<Forecast>,
		lastSmartCostLimit: Number,
		lastSmartFeedInPriorityLimit: Number,
	},
	data() {
		return {
			tickerHandler: null as Timeout,
			phaseRemainingInterpolated: this.phaseRemaining,
			pvRemainingInterpolated: this.pvRemaining,
			chargeDurationInterpolated: this.chargeDuration,
			chargeRemainingDurationInterpolated: this.chargeRemainingDuration,
			loadpointViewportMaximized: false,
			bodyOverflowBefore: null as string | null,
		};
	},
	computed: {
		loadpointRootAttrs() {
			return this.$attrs;
		},
		expandLoadpointHeader() {
			return this.multipleLoadpoints && !this.fullWidth;
		},
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
			return this.batteryConfigured;
		},
		batteryBoostActive() {
			return (
				this.batteryBoost &&
				this.charging &&
				this.mode &&
				!["off", "now"].includes(this.mode) &&
				(this.batterySoc ?? 0) >= this.batteryBoostLimit
			);
		},
		plannerForecast() {
			return this.forecast?.planner;
		},
		shouldMaximizeFromRoute(): boolean {
			return this.$route?.query?.maximize === this.id;
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
		loadpointViewportMaximized(expanded: boolean) {
			if (expanded) {
				this.bodyOverflowBefore = document.body.style.overflow;
				document.body.style.overflow = "hidden";
			} else {
				document.body.style.overflow = this.bodyOverflowBefore ?? "";
				this.bodyOverflowBefore = null;
			}
		},
		shouldMaximizeFromRoute: {
			handler(shouldMaximize: boolean) {
				if (shouldMaximize && !this.loadpointViewportMaximized) {
					this.loadpointViewportMaximized = true;
				}
			},
			immediate: true,
		},
	},
	mounted() {
		this.tickerHandler = setInterval(this.tick, 1000);
		window.addEventListener("keydown", this.handleViewportEscape);
	},
	unmounted() {
		if (this.tickerHandler) {
			clearInterval(this.tickerHandler);
		}
		window.removeEventListener("keydown", this.handleViewportEscape);
		if (this.loadpointViewportMaximized) {
			document.body.style.overflow = this.bodyOverflowBefore ?? "";
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
		setBatteryBoostLimit(limit: number) {
			api.post(this.apiPath("batteryboostlimit") + "/" + limit);
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
		expandViewport() {
			this.loadpointViewportMaximized = true;
			const query = { ...this.$route.query, maximize: this.id };
			this.$router.replace({ query });
		},
		collapseViewport() {
			this.loadpointViewportMaximized = false;
			const query = { ...this.$route.query };
			delete query.maximize;
			this.$router.replace({ query });
		},
		handleViewportEscape(ev: KeyboardEvent) {
			if (ev.key !== "Escape" || !this.loadpointViewportMaximized) {
				return;
			}
			if (document.querySelector(".modal.show")) {
				return;
			}
			this.collapseViewport();
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

.loadpoint-inline-host {
	display: flex;
	flex-direction: column;
	height: 100%;
	min-height: 0;
}

.loadpoint-viewport-overlay {
	position: fixed;
	inset: 0;
	z-index: 1030;
	display: flex;
	flex-direction: column;
	min-height: 100dvh;
	min-height: 100vh;
	background: var(--evcc-background);
	overflow: hidden;
}

.loadpoint {
	border-radius: 2rem;
	color: var(--evcc-default-text);
	background: var(--evcc-box);
}

.loadpoint--viewport-expanded {
	min-height: 0;
	flex: 1 1 auto;
	overflow: auto;
}

.loadpoint-header-wide__title {
	flex: 1 1 8rem;
	min-width: 0;
}

.loadpoint-expand-icon {
	display: block;
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
@media (--sm-and-up) {
	.divider {
		margin: 0 -1.5rem;
	}
}
</style>
