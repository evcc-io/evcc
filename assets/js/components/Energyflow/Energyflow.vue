<template>
	<div
		class="energyflow cursor-pointer position-relative"
		:class="{ 'energyflow--open': detailsOpen }"
		data-testid="energyflow"
		@click="toggleDetails"
	>
		<div class="row">
			<Visualization
				class="col-12 mb-3 mb-md-4"
				:gridImport="gridImport"
				:selfPv="selfPv"
				:selfBattery="selfBattery"
				:loadpoints="loadpointsCompact"
				:pvExport="pvExport"
				:batteryCharge="batteryCharge"
				:batteryDischarge="batteryDischarge"
				:batteryGridCharge="batteryGridChargeActive"
				:batteryHold="batteryHold"
				:pvProduction="pvProduction"
				:homePower="homePower"
				:batterySoc="batterySoc"
				:powerUnit="powerUnit"
				:vehicleIcons="vehicleIcons"
				:inPower="inPower"
				:outPower="outPower"
			/>
		</div>
		<div
			class="details"
			:style="{ height: detailsHeight }"
			:class="{ 'details--ready': ready }"
		>
			<div ref="detailsInner" class="details-inner row">
				<div class="col-12 d-flex justify-content-between pt-2 mb-4">
					<div class="d-flex flex-nowrap align-items-center text-truncate">
						<span class="me-2 legend-self"
							><shopicon-filled-square
								class="color-pv legend-pv"
							></shopicon-filled-square>
							<shopicon-filled-square
								v-if="selfBattery > 0"
								:class="`color-battery legend-battery legend-battery--${selfPv > 0 ? 'mixed' : 'only'}`"
							></shopicon-filled-square
						></span>
						<span class="text-nowrap text-truncate">
							{{ $t("main.energyflow.selfConsumption") }}
						</span>
					</div>
					<div
						v-if="gridImport > 0"
						class="d-flex flex-nowrap align-items-center text-truncate"
					>
						<span class="text-nowrap text-truncate">
							{{ $t("main.energyflow.gridImport") }}
						</span>
						<span class="ms-2"
							><shopicon-filled-square class="legend-grid"></shopicon-filled-square
						></span>
					</div>
					<div v-else class="d-flex flex-nowrap align-items-center text-truncate">
						<span class="text-nowrap text-truncate">
							{{ $t("main.energyflow.pvExport") }}
						</span>
						<span class="ms-2"
							><shopicon-filled-square class="legend-export"></shopicon-filled-square
						></span>
					</div>
				</div>
				<div
					class="col-12 col-md-6 pe-md-5 pb-4 d-flex flex-column justify-content-between"
				>
					<div class="d-flex justify-content-between align-items-end mb-4">
						<h3 class="m-0">In</h3>
						<span v-if="pvPossible" class="fw-bold">
							<AnimatedNumber :to="inPower" :format="kw" />
						</span>
					</div>
					<div>
						<EnergyflowEntry
							v-if="pvPossible"
							:name="$t('main.energyflow.pvProduction')"
							icon="sun"
							:power="pvProduction"
							:powerTooltip="pvTooltip"
							:powerUnit="powerUnit"
							data-testid="energyflow-entry-production"
						/>
						<EnergyflowEntry
							v-if="batteryConfigured"
							:name="batteryDischargeLabel"
							icon="battery"
							:power="batteryDischarge"
							:powerUnit="powerUnit"
							:iconProps="{
								hold: batteryHold,
								soc: batterySoc,
								gridCharge: batteryGridChargeActive,
							}"
							:details="batterySoc"
							:detailsFmt="batteryFmt"
							detailsClickable
							data-testid="energyflow-entry-batterydischarge"
							@details-clicked="openBatterySettingsModal"
						>
							<template v-if="batteryGridChargeLimitSet" #subline>
								<div class="d-none d-md-block">&nbsp;</div>
							</template>
						</EnergyflowEntry>
						<EnergyflowEntry
							:name="$t('main.energyflow.gridImport')"
							icon="powersupply"
							:power="gridImport"
							:powerUnit="powerUnit"
							:details="detailsValue(tariffGrid, tariffCo2)"
							:detailsFmt="detailsFmt"
							:detailsTooltip="detailsTooltip(tariffGrid, tariffCo2)"
							data-testid="energyflow-entry-gridimport"
						/>
					</div>
				</div>
				<div
					class="col-12 col-md-6 ps-md-5 pb-4 d-flex flex-column justify-content-between"
				>
					<div class="d-flex justify-content-between align-items-end mb-4">
						<h3 class="m-0">Out</h3>
						<span v-if="pvPossible" class="fw-bold">
							<AnimatedNumber :to="outPower" :format="kw" />
						</span>
					</div>
					<div>
						<EnergyflowEntry
							v-if="pvPossible"
							:name="$t('main.energyflow.homePower')"
							icon="home"
							:power="homePower"
							:powerUnit="powerUnit"
							:details="detailsValue(tariffPriceHome, tariffCo2Home)"
							:detailsFmt="detailsFmt"
							:detailsTooltip="detailsTooltip(tariffPriceHome, tariffCo2Home)"
							data-testid="energyflow-entry-home"
						/>
						<EnergyflowEntry
							:name="
								$t('main.energyflow.loadpoints', activeLoadpointsCount, {
									count: activeLoadpointsCount,
								})
							"
							icon="vehicle"
							:iconProps="{ names: vehicleIcons }"
							:power="loadpointsPower"
							:powerUnit="powerUnit"
							:details="
								activeLoadpointsCount
									? detailsValue(tariffPriceLoadpoints, tariffCo2Loadpoints)
									: undefined
							"
							:detailsFmt="detailsFmt"
							:detailsTooltip="
								detailsTooltip(tariffPriceLoadpoints, tariffCo2Loadpoints)
							"
							data-testid="energyflow-entry-loadpoints"
						/>
						<EnergyflowEntry
							v-if="batteryConfigured"
							:name="batteryChargeLabel"
							icon="battery"
							:power="batteryCharge"
							:powerUnit="powerUnit"
							:iconProps="{
								hold: batteryHold,
								soc: batterySoc,
								gridCharge: batteryGridChargeActive,
							}"
							:details="batterySoc"
							:detailsFmt="batteryFmt"
							detailsClickable
							@details-clicked="openBatterySettingsModal"
						>
							<template v-if="batteryGridChargeLimitSet" #subline>
								<button
									type="button"
									class="btn-reset d-flex justify-content-between text-start pe-4"
									@click.stop="openBatterySettingsModal"
								>
									<span v-if="batteryGridChargeActive">
										{{ $t("main.energyflow.batteryGridChargeActive") }}
										<span class="text-nowrap"
											>(≤ <u>{{ batteryGridChargeLimitFmt }}</u
											>)</span
										>
									</span>
									<span v-else>
										{{ $t("main.energyflow.batteryGridChargeLimit") }}
										<span class="text-nowrap"
											>≤ <u>{{ batteryGridChargeLimitFmt }}</u></span
										>
									</span>
								</button>
							</template>
						</EnergyflowEntry>
						<EnergyflowEntry
							v-if="pvPossible"
							:name="$t('main.energyflow.pvExport')"
							icon="powersupply"
							:power="pvExport"
							:powerUnit="powerUnit"
							:details="detailsValue(-tariffFeedIn)"
							:detailsFmt="detailsFmt"
							:detailsTooltip="detailsTooltip(-tariffFeedIn)"
							data-testid="energyflow-entry-gridexport"
						/>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/square";
import Modal from "bootstrap/js/dist/modal";
import Visualization from "./Visualization.vue";
import EnergyflowEntry from "./EnergyflowEntry.vue";
import formatter, { POWER_UNIT } from "../../mixins/formatter";
import AnimatedNumber from "../AnimatedNumber.vue";
import settings from "../../settings";
import { CO2_TYPE } from "../../units";
import collector from "../../mixins/collector";

export default {
	name: "Energyflow",
	components: {
		Visualization,
		EnergyflowEntry,
		AnimatedNumber,
	},
	mixins: [formatter, collector],
	props: {
		gridConfigured: Boolean,
		gridPower: { type: Number, default: 0 },
		homePower: { type: Number, default: 0 },
		pvConfigured: Boolean,
		pv: { type: Array },
		pvPower: { type: Number, default: 0 },
		loadpointsCompact: { type: Array, default: () => [] },
		batteryConfigured: { type: Boolean },
		battery: { type: Array },
		batteryPower: { type: Number, default: 0 },
		batterySoc: { type: Number, default: 0 },
		batteryDischargeControl: { type: Boolean },
		batteryGridChargeLimit: { type: Number },
		batteryGridChargeActive: { type: Boolean },
		batteryMode: { type: String },
		tariffGrid: { type: Number },
		tariffFeedIn: { type: Number },
		tariffCo2: { type: Number },
		tariffPriceHome: { type: Number },
		tariffCo2Home: { type: Number },
		tariffPriceLoadpoints: { type: Number },
		tariffCo2Loadpoints: { type: Number },
		smartCostType: { type: String },
		currency: { type: String },
		prioritySoc: { type: Number },
		bufferSoc: { type: Number },
		bufferStartSoc: { type: Number },
	},
	data: () => {
		return { detailsOpen: false, detailsCompleteHeight: null, ready: false };
	},
	computed: {
		gridImport: function () {
			return Math.max(0, this.gridPower);
		},
		pvProduction: function () {
			return Math.abs(this.pvPower);
		},
		batteryDischarge: function () {
			return Math.abs(Math.max(0, this.batteryPower));
		},
		batteryCharge: function () {
			return Math.abs(Math.min(0, this.batteryPower) * -1);
		},
		batteryChargeLabel: function () {
			return this.$t(`main.energyflow.battery${this.batteryHold ? "Hold" : "Charge"}`);
		},
		batteryDischargeLabel: function () {
			return this.$t(`main.energyflow.battery${this.batteryHold ? "Hold" : "Discharge"}`);
		},
		batteryHold: function () {
			return this.batteryMode === "hold";
		},
		consumption: function () {
			return this.homePower + this.batteryCharge + this.loadpointsPower;
		},
		selfPv: function () {
			return Math.min(this.pvProduction, this.consumption);
		},
		selfBattery: function () {
			return Math.min(this.batteryDischarge, this.consumption - this.selfPv);
		},
		activeLoadpoints: function () {
			return this.loadpointsCompact.filter((lp) => lp.charging);
		},
		activeLoadpointsCount: function () {
			return this.activeLoadpoints.length;
		},
		vehicleIcons: function () {
			if (this.activeLoadpointsCount > 0) {
				return this.activeLoadpoints.map((lp) => lp.icon);
			}
			return ["car"];
		},
		loadpointsPower: function () {
			return this.loadpointsCompact.reduce((sum, lp) => {
				return sum + (lp.power || 0);
			}, 0);
		},
		pvExport: function () {
			return Math.max(0, this.gridPower * -1);
		},
		powerUnit: function () {
			const watt = Math.max(this.gridImport, this.selfPv, this.selfBattery, this.pvExport);
			if (watt >= 1_000_000) {
				return POWER_UNIT.MW;
			} else if (watt >= 1000) {
				return POWER_UNIT.KW;
			} else {
				return POWER_UNIT.W;
			}
		},
		inPower: function () {
			return this.gridImport + this.pvProduction + this.batteryDischarge;
		},
		outPower: function () {
			return this.homePower + this.loadpointsPower + this.pvExport + this.batteryCharge;
		},
		detailsHeight: function () {
			return this.detailsOpen ? this.detailsCompleteHeight + "px" : 0;
		},
		pvTooltip() {
			if (!Array.isArray(this.pv) || this.pv.length <= 1) {
				return;
			}
			return this.pv.map(({ power }) => this.fmtW(power, this.powerUnit));
		},
		batteryFmt() {
			return (soc) => this.fmtPercentage(soc, 0);
		},
		co2Available() {
			return this.smartCostType === CO2_TYPE;
		},
		pvPossible() {
			return this.pvConfigured || this.gridConfigured;
		},
		batteryGridChargeNow() {
			if (this.co2Available) {
				return this.fmtCo2Short(this.tariffCo2);
			}
			return this.fmtPricePerKWh(this.tariffGrid, this.currency, true);
		},
		batteryGridChargeLimitSet() {
			return this.batteryGridChargeLimit !== null;
		},
		batteryGridChargeLimitFmt() {
			if (!this.batteryGridChargeLimitSet) {
				return;
			}
			if (this.co2Available) {
				return this.fmtCo2Short(this.batteryGridChargeLimit);
			}
			return this.fmtPricePerKWh(this.batteryGridChargeLimit, this.currency, true);
		},
	},
	watch: {
		pvConfigured() {
			this.$nextTick(this.updateHeight);
		},
		gridConfigured() {
			this.$nextTick(this.updateHeight);
		},
		batteryConfigured() {
			this.$nextTick(this.updateHeight);
		},
		batteryMode() {
			this.$nextTick(this.updateHeight);
		},
	},
	mounted() {
		window.addEventListener("resize", this.updateHeight);
		// height must be calculated in case of initially open details
		if (settings.energyflowDetails) {
			this.toggleDetails();
		}
		setTimeout(() => (this.ready = true), 200);
	},
	unmounted() {
		window.removeEventListener("resize", this.updateHeight);
	},
	methods: {
		detailsTooltip(price, co2) {
			const result = [];
			if (co2 !== undefined) {
				result.push(`${this.fmtCo2Long(co2)}`);
			}
			if (price !== undefined) {
				result.push(`${this.fmtPricePerKWh(price, this.currency)}`);
			}
			return result;
		},
		detailsValue(price, co2) {
			if (this.co2Available) {
				return co2;
			}
			return price;
		},
		detailsFmt(value) {
			if (this.co2Available) {
				return this.fmtCo2Short(value);
			}
			return this.fmtPricePerKWh(value, this.currency, true);
		},
		kw: function (watt) {
			return this.fmtW(watt, this.powerUnit);
		},
		toggleDetails: function () {
			this.updateHeight();
			this.detailsOpen = !this.detailsOpen;
			settings.energyflowDetails = this.detailsOpen;
		},
		updateHeight: function () {
			this.detailsCompleteHeight = this.$refs.detailsInner.offsetHeight;
		},
		openBatterySettingsModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("batterySettingsModal")
			);
			modal.show();
		},
	},
};
</script>
<style scoped>
.details {
	height: 0;
	opacity: 0;
	transform: scale(0.98);
	overflow: visible;
	transition-property: height, opacity, transform;
	transition-duration: 0;
	transition-timing-function: cubic-bezier(0.5, 0.5, 0.5, 1.15);
}
.details--ready {
	transition-duration: var(--evcc-transition-medium);
}
.energyflow--open .details {
	opacity: 1;
	transform: scale(1);
}
.color-grid {
	color: var(--evcc-grid);
}
.color-export {
	color: var(--evcc-export);
}
.legend-grid {
	color: var(--evcc-grid);
}
.legend-export {
	color: var(--evcc-export);
}
.legend-pv {
	color: var(--evcc-pv);
}
.legend-self {
	position: relative;
}
.legend-battery {
	position: absolute;
	top: 0;
	left: 0;
	color: var(--evcc-battery);
}
.legend-battery--mixed {
	clip-path: polygon(100% 0, 100% 100%, 0 100%);
}
</style>
