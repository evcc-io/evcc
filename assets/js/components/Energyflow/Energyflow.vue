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
				:selfConsumption="selfConsumption"
				:loadpoints="loadpointsCompact"
				:pvExport="pvExport"
				:batteryCharge="batteryCharge"
				:batteryDischarge="batteryDischarge"
				:pvProduction="pvProduction"
				:homePower="homePower"
				:batterySoc="batterySoc"
				:powerInKw="powerInKw"
				:vehicleIcons="vehicleIcons"
			/>
		</div>
		<div class="details" :style="{ height: detailsHeight }">
			<div ref="detailsInner" class="details-inner row">
				<div class="col-12 d-flex justify-content-between pt-2 mb-4">
					<div class="d-flex flex-nowrap align-items-center text-truncate">
						<span class="color-self me-2"
							><shopicon-filled-square></shopicon-filled-square
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
						<span class="color-grid ms-2"
							><shopicon-filled-square></shopicon-filled-square
						></span>
					</div>
					<div v-else class="d-flex flex-nowrap align-items-center text-truncate">
						<span class="text-nowrap text-truncate">
							{{ $t("main.energyflow.pvExport") }}
						</span>
						<span class="color-export ms-2"
							><shopicon-filled-square></shopicon-filled-square
						></span>
					</div>
				</div>
				<div
					class="col-12 col-md-6 pe-md-5 pb-4 d-flex flex-column justify-content-between"
				>
					<div class="d-flex justify-content-between align-items-end mb-4">
						<h3 class="m-0">In</h3>
						<span class="fw-bold">
							<AnimatedNumber :to="inPower" :format="kw" />
						</span>
					</div>
					<div>
						<EnergyflowEntry
							:name="$t('main.energyflow.pvProduction')"
							icon="sun"
							:power="pvProduction"
							:powerTooltip="pvTooltip"
							:powerInKw="powerInKw"
						/>
						<EnergyflowEntry
							v-if="batteryConfigured"
							:name="batteryDischargeLabel"
							icon="battery"
							:power="batteryDischarge"
							:powerInKw="powerInKw"
							:soc="batterySoc"
							:details="batterySoc"
							:detailsFmt="batteryFmt"
							detailsClickable
							data-testid="energyflow-entry-batterydischarge"
							@details-clicked="openBatterySettingsModal"
						/>
						<EnergyflowEntry
							:name="$t('main.energyflow.gridImport')"
							icon="powersupply"
							:power="gridImport"
							:powerInKw="powerInKw"
							:details="detailsValue(tariffGrid, tariffCo2)"
							:detailsFmt="detailsFmt"
							:detailsClickable="gridModalAvailable"
							:detailsTooltip="detailsTooltip(tariffGrid, tariffCo2)"
							data-testid="energyflow-entry-gridimport"
							@details-clicked="openGridSettingsModal"
						/>
					</div>
				</div>
				<div
					class="col-12 col-md-6 ps-md-5 pb-4 d-flex flex-column justify-content-between"
				>
					<div class="d-flex justify-content-between align-items-end mb-4">
						<h3 class="m-0">Out</h3>
						<span class="fw-bold">
							<AnimatedNumber :to="outPower" :format="kw" />
						</span>
					</div>
					<div>
						<EnergyflowEntry
							:name="$t('main.energyflow.homePower')"
							icon="home"
							:power="homePower"
							:powerInKw="powerInKw"
							:details="detailsValue(tariffPriceHome, tariffCo2Home)"
							:detailsFmt="detailsFmt"
							:detailsTooltip="detailsTooltip(tariffPriceHome, tariffCo2Home)"
						/>
						<EnergyflowEntry
							:name="
								$tc('main.energyflow.loadpoints', activeLoadpointsCount, {
									count: activeLoadpointsCount,
								})
							"
							icon="vehicle"
							:vehicleIcons="vehicleIcons"
							:power="loadpointsPower"
							:powerInKw="powerInKw"
							:details="
								activeLoadpointsCount
									? detailsValue(tariffPriceLoadpoints, tariffCo2Loadpoints)
									: undefined
							"
							:detailsFmt="detailsFmt"
							:detailsTooltip="
								detailsTooltip(tariffPriceLoadpoints, tariffCo2Loadpoints)
							"
						/>
						<EnergyflowEntry
							v-if="batteryConfigured"
							:name="batteryChargeLabel"
							icon="battery"
							:power="batteryCharge"
							:powerInKw="powerInKw"
							:soc="batterySoc"
							:details="batterySoc"
							:detailsFmt="batteryFmt"
							detailsClickable
							@details-clicked="openBatterySettingsModal"
						/>
						<EnergyflowEntry
							:name="$t('main.energyflow.pvExport')"
							icon="powersupply"
							:power="pvExport"
							:powerInKw="powerInKw"
							:details="detailsValue(-tariffFeedIn)"
							:detailsFmt="detailsFmt"
							:detailsTooltip="detailsTooltip(-tariffFeedIn)"
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
import formatter from "../../mixins/formatter";
import AnimatedNumber from "../AnimatedNumber.vue";
import settings from "../../settings";
import { CO2_TYPE } from "../../units";
import collector from "../../mixins/collector";
import gridModalAvailable from "../../utils/gridModalAvailable";

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
		batteryMode: { type: String },
		tariffGrid: { type: Number },
		tariffFeedIn: { type: Number },
		tariffCo2: { type: Number },
		tariffPriceHome: { type: Number },
		tariffCo2Home: { type: Number },
		tariffPriceLoadpoints: { type: Number },
		tariffCo2Loadpoints: { type: Number },
		smartCostLimit: { type: Number },
		smartCostType: { type: String },
		currency: { type: String },
		prioritySoc: { type: Number },
		bufferSoc: { type: Number },
		bufferStartSoc: { type: Number },
	},
	data: () => {
		return { detailsOpen: false, detailsCompleteHeight: null, gridSettingsModal: null };
	},
	computed: {
		gridModalAvailable: function () {
			return gridModalAvailable(this.smartCostType);
		},
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
		selfConsumption: function () {
			const ownPower = this.batteryDischarge + this.pvProduction;
			const consumption = this.homePower + this.batteryCharge + this.loadpointsPower;
			return Math.min(ownPower, consumption);
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
				sum += lp.power || 0;
				return sum;
			}, 0);
		},
		pvExport: function () {
			return Math.max(0, this.gridPower * -1);
		},
		powerInKw: function () {
			return Math.max(this.gridImport, this.selfConsumption, this.pvExport) >= 1000;
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
			return this.pv.map(({ power }) => this.fmtKw(power, this.powerInKw));
		},
		batteryFmt() {
			return (soc) => `${Math.round(soc)}%`;
		},
		co2Available() {
			return this.smartCostType === CO2_TYPE;
		},
	},
	mounted() {
		window.addEventListener("resize", this.updateHeight);
		// height must be calculated in case of initially open details
		if (settings.energyflowDetails) {
			setTimeout(this.toggleDetails, 100);
		}
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
			return this.fmtKw(watt, this.powerInKw);
		},
		toggleDetails: function () {
			this.updateHeight();
			this.detailsOpen = !this.detailsOpen;
			settings.energyflowDetails = this.detailsOpen;
		},
		updateHeight: function () {
			this.detailsCompleteHeight = this.$refs.detailsInner.offsetHeight;
		},
		openGridSettingsModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("gridSettingsModal"));
			modal.show();
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
	transition: height, opacity, transform;
	transition-duration: var(--evcc-transition-medium);
	transition-timing-function: cubic-bezier(0.5, 0.5, 0.5, 1.15);
}
.energyflow--open .details {
	opacity: 1;
	transform: scale(1);
}
.color-grid {
	color: var(--evcc-grid);
}
.color-self {
	color: var(--evcc-self);
}
.color-export {
	color: var(--evcc-export);
}
</style>
