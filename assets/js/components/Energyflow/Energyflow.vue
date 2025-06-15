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
					<div class="d-flex justify-content-between align-items-baseline mb-4">
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
							:details="solarForecastRemainingToday"
							:detailsFmt="forecastFmt"
							:detailsTooltip="solarForecastTooltip"
							:detailsInactive="!solarForecastExists"
							:detailsIcon="solarForecastIcon"
							:detailsClickable="solarForecastExists"
							:powerUnit="powerUnit"
							:expanded="pvExpanded"
							data-testid="energyflow-entry-production"
							@details-clicked="openForecastModal"
							@toggle="togglePv"
						>
							<template v-if="multiplePv" #expanded>
								<EnergyflowEntry
									v-for="(p, index) in pv"
									:key="index"
									:name="p.title || genericPvTitle(index)"
									:power="p.power"
									:powerUnit="powerUnit"
									:data-testid="`energyflow-entry-production-${index}`"
								/>
							</template>
						</EnergyflowEntry>
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
							:expanded="batteryExpanded"
							detailsClickable
							data-testid="energyflow-entry-batterydischarge"
							@details-clicked="openBatterySettingsModal"
							@toggle="toggleBattery"
						>
							<template v-if="batteryGridChargeLimitSet" #subline>
								<div class="d-none d-md-block">&nbsp;</div>
							</template>
							<template v-if="multipleBattery" #expanded>
								<EnergyflowEntry
									v-for="(b, index) in battery"
									:key="index"
									:name="b.title || genericBatteryTitle(index)"
									:details="b.soc"
									:detailsFmt="batteryFmt"
									:power="dischargePower(b.power)"
									:powerUnit="powerUnit"
								/>
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
					<div class="d-flex justify-content-between align-items-baseline mb-4">
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
								// @ts-ignore
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
							:expanded="loadpointsExpanded"
							@toggle="toggleLoadpoints"
						>
							<template v-if="activeLoadpointsCount > 0" #expanded>
								<EnergyflowEntry
									v-for="lp in activeLoadpoints"
									:key="lp.index"
									:name="lp.title"
									:power="lp.power"
									:powerUnit="powerUnit"
									icon="vehicle"
									:iconProps="{ names: [lp.icon] }"
									:details="lp.soc || undefined"
									:detailsFmt="lp.heating ? fmtLoadpointTemp : fmtLoadpointSoc"
								/>
							</template>
						</EnergyflowEntry>
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
							:expanded="batteryExpanded"
							detailsClickable
							@details-clicked="openBatterySettingsModal"
							@toggle="toggleBattery"
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
							<template v-if="multipleBattery" #expanded>
								<EnergyflowEntry
									v-for="(b, index) in battery"
									:key="index"
									:name="b.title || genericBatteryTitle(index)"
									:details="b.soc"
									:detailsFmt="batteryFmt"
									:power="chargePower(b.power)"
									:powerUnit="powerUnit"
								/>
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

<script lang="ts">
import "@h2d2/shopicons/es/filled/square";
import Modal from "bootstrap/js/dist/modal";
import Visualization from "./Visualization.vue";
import Entry from "./Entry.vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import settings from "@/settings";
import collector from "@/mixins/collector.js";
import { defineComponent, type PropType } from "vue";
import {
	SMART_COST_TYPE,
	type Battery,
	type CURRENCY,
	type Forecast,
	type LoadpointCompact,
} from "@/types/evcc";

export default defineComponent({
	name: "Energyflow",
	components: {
		Visualization,
		EnergyflowEntry: Entry,
		AnimatedNumber,
	},
	mixins: [formatter, collector],
	props: {
		gridConfigured: Boolean,
		gridPower: { type: Number, default: 0 },
		homePower: { type: Number, default: 0 },
		pvConfigured: Boolean,
		pv: { type: Array as PropType<Pv[]> },
		pvPower: { type: Number, default: 0 },
		loadpointsCompact: { type: Array as PropType<LoadpointCompact[]>, default: () => [] },
		batteryConfigured: { type: Boolean },
		battery: { type: Array as PropType<Battery[]> },
		batteryPower: { type: Number, default: 0 },
		batterySoc: { type: Number, default: 0 },
		batteryDischargeControl: { type: Boolean },
		batteryGridChargeLimit: { type: Number },
		batteryGridChargeActive: { type: Boolean },
		batteryMode: { type: String },
		tariffGrid: { type: Number },
		tariffFeedIn: { type: Number, default: 0 },
		tariffCo2: { type: Number },
		tariffPriceHome: { type: Number },
		tariffCo2Home: { type: Number },
		tariffPriceLoadpoints: { type: Number },
		tariffCo2Loadpoints: { type: Number },
		smartCostType: { type: String },
		currency: { type: String as PropType<CURRENCY> },
		prioritySoc: { type: Number },
		bufferSoc: { type: Number },
		bufferStartSoc: { type: Number },
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
	},
	data: () => {
		return { detailsOpen: false, detailsCompleteHeight: null as number | null, ready: false };
	},
	computed: {
		gridImport() {
			return Math.max(0, this.gridPower);
		},
		pvProduction() {
			return Math.abs(this.pvPower);
		},
		batteryDischarge() {
			return this.dischargePower(this.batteryPower);
		},
		batteryCharge() {
			return this.chargePower(this.batteryPower);
		},
		batteryChargeLabel() {
			return this.$t(`main.energyflow.battery${this.batteryHold ? "Hold" : "Charge"}`);
		},
		batteryDischargeLabel() {
			return this.$t(`main.energyflow.battery${this.batteryHold ? "Hold" : "Discharge"}`);
		},
		batteryHold() {
			return this.batteryMode === "hold";
		},
		consumption() {
			return this.homePower + this.batteryCharge + this.loadpointsPower;
		},
		selfPv() {
			return Math.min(this.pvProduction, this.consumption);
		},
		selfBattery() {
			return Math.min(this.batteryDischarge, this.consumption - this.selfPv);
		},
		activeLoadpoints() {
			return this.loadpointsCompact.filter((lp) => lp.charging);
		},
		activeLoadpointsCount() {
			return this.activeLoadpoints.length;
		},
		vehicleIcons() {
			if (this.activeLoadpointsCount > 0) {
				return this.activeLoadpoints.map((lp) => lp.icon);
			}
			return ["car"];
		},
		loadpointsPower() {
			return this.loadpointsCompact.reduce((sum, lp) => {
				return sum + (lp.power || 0);
			}, 0);
		},
		pvExport() {
			return Math.max(0, this.gridPower * -1);
		},
		powerUnit() {
			const watt = Math.max(this.gridImport, this.selfPv, this.selfBattery, this.pvExport);
			if (watt >= 1_000_000) {
				return POWER_UNIT.MW;
			} else if (watt >= 1000) {
				return POWER_UNIT.KW;
			} else {
				return POWER_UNIT.W;
			}
		},
		inPower() {
			return this.gridImport + this.pvProduction + this.batteryDischarge;
		},
		outPower() {
			return this.homePower + this.loadpointsPower + this.pvExport + this.batteryCharge;
		},
		detailsHeight() {
			return this.detailsOpen ? this.detailsCompleteHeight + "px" : 0;
		},
		batteryFmt() {
			return (soc: number) => this.fmtPercentage(soc, 0);
		},
		multipleBattery() {
			return (this.battery?.length || 0) > 1;
		},
		multiplePv() {
			return (this.pv?.length || 0) > 1;
		},
		fmtLoadpointSoc() {
			return (soc: number) => this.fmtPercentage(soc, 0);
		},
		fmtLoadpointTemp() {
			return (temp: number) => this.fmtTemperature(temp);
		},
		co2Available() {
			return this.smartCostType === SMART_COST_TYPE.CO2;
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
			return (
				this.batteryGridChargeLimit !== null && this.batteryGridChargeLimit !== undefined
			);
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
		solarForecastExists() {
			return !!this.forecast?.solar;
		},
		solarForecastRemainingToday() {
			if (!this.forecast?.solar) {
				return undefined;
			}
			const { today, scale } = this.forecast.solar || {};
			const factor = this.$hiddenFeatures() && settings.solarAdjusted && scale ? scale : 1;
			const energy = today?.energy || 0;
			return energy * factor;
		},
		solarForecastIcon() {
			return this.solarForecastExists ? "forecast" : undefined;
		},
		solarForecastTooltip() {
			if (this.solarForecastExists) {
				return [this.$t("main.energyflow.forecastTooltip")];
			}
			return [];
		},
		pvExpanded() {
			return settings.energyflowPv;
		},
		batteryExpanded() {
			return settings.energyflowBattery;
		},
		loadpointsExpanded() {
			return settings.energyflowLoadpoints;
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
		activeLoadpointsCount() {
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
		detailsTooltip(price?: number, co2?: number) {
			const result = [];
			if (co2 !== undefined) {
				result.push(`${this.fmtCo2Long(co2)}`);
			}
			if (price !== undefined) {
				result.push(`${this.fmtPricePerKWh(price, this.currency)}`);
			}
			return result;
		},
		detailsValue(price?: number, co2?: number) {
			if (this.co2Available) {
				return co2;
			}
			return price;
		},
		detailsFmt(value: number) {
			if (this.co2Available) {
				return this.fmtCo2Short(value);
			}
			return this.fmtPricePerKWh(value, this.currency, true);
		},
		forecastFmt(value: number) {
			if (typeof value !== "number") return "";
			return `${this.fmtWh(value, POWER_UNIT.KW)}`;
		},
		kw(watt: number) {
			if (typeof watt !== "number") return "";
			return this.fmtW(watt, this.powerUnit);
		},
		toggleDetails() {
			this.updateHeight();
			this.detailsOpen = !this.detailsOpen;
			settings.energyflowDetails = this.detailsOpen;
		},
		updateHeight() {
			this.detailsCompleteHeight = this.$refs["detailsInner"]?.offsetHeight ?? 0;
		},
		openBatterySettingsModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("batterySettingsModal") as HTMLElement
			);
			modal.show();
		},
		openForecastModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("forecastModal") as HTMLElement
			);
			modal.show();
		},
		dischargePower(power: number) {
			return Math.abs(Math.max(0, power));
		},
		chargePower(power: number) {
			return Math.abs(Math.min(0, power) * -1);
		},
		toggleBattery() {
			settings.energyflowBattery = !settings.energyflowBattery;
			this.$nextTick(this.updateHeight);
		},
		togglePv() {
			settings.energyflowPv = !settings.energyflowPv;
			this.$nextTick(this.updateHeight);
		},
		toggleLoadpoints() {
			settings.energyflowLoadpoints = !settings.energyflowLoadpoints;
			this.$nextTick(this.updateHeight);
		},
		genericBatteryTitle(index: number) {
			return `${this.$t("config.devices.batteryStorage")} #${index + 1}`;
		},
		genericPvTitle(index: number) {
			return `${this.$t("config.devices.solarSystem")} #${index + 1}`;
		},
	},
});
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
