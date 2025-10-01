<template>
	<div
		data-testid="visualization"
		class="visualization"
		:class="{ 'visualization--ready': visualizationReady }"
	>
		<div class="label-scale d-flex">
			<div class="d-flex justify-content-start flex-grow-1">
				<LabelBar v-bind="labelBarProps('top', 'pvProduction')">
					<shopicon-regular-sun></shopicon-regular-sun>
				</LabelBar>
				<LabelBar v-bind="labelBarProps('top', 'batteryDischarge')">
					<BatteryIcon :soc="batterySoc" />
				</LabelBar>
				<LabelBar v-bind="labelBarProps('top', 'gridImport')">
					<shopicon-regular-powersupply></shopicon-regular-powersupply>
				</LabelBar>
				<LabelBar v-bind="labelBarProps('top', 'unknownImport')">
					<QuestionIcon />
				</LabelBar>
			</div>
			<div class="label-scale-name">In</div>
		</div>
		<div ref="site_progress" class="site-progress">
			<div class="site-progress-bar self-pv" :style="{ width: widthTotal(selfPvAdjusted) }">
				<AnimatedNumber
					v-if="selfPv && visualizationReady"
					class="power"
					:to="selfPv"
					:format="fmtBarValue"
				/>
			</div>
			<div
				class="site-progress-bar self-battery"
				:style="{ width: widthTotal(selfBatteryAdjusted) }"
			>
				<AnimatedNumber
					v-if="selfBattery && visualizationReady"
					class="power"
					:to="selfBattery"
					:format="fmtBarValue"
				/>
			</div>
			<div
				class="site-progress-bar grid-import"
				:style="{ width: widthTotal(gridImportAdjusted) }"
			>
				<AnimatedNumber
					v-if="gridImport && visualizationReady"
					class="power"
					:to="gridImport"
					:format="fmtBarValue"
				/>
			</div>
			<div
				class="site-progress-bar pv-export"
				:style="{ width: widthTotal(pvExportAdjusted) }"
			>
				<AnimatedNumber
					v-if="pvExport && visualizationReady"
					class="power"
					:to="pvExport"
					:format="fmtBarValue"
				/>
			</div>
			<div
				class="site-progress-bar unknown-power"
				:style="{ width: widthTotal(unknownPower) }"
			>
				<AnimatedNumber
					v-if="unknownPower && visualizationReady"
					class="power"
					:to="unknownPower"
					:format="fmtBarValue"
				/>
			</div>
			<div v-if="totalAdjusted <= 0" class="site-progress-bar w-100 grid-import">
				<span>{{ fmtW(0, POWER_UNIT.AUTO, true) }}</span>
			</div>
		</div>
		<div class="label-scale d-flex">
			<div class="d-flex justify-content-start flex-grow-1">
				<LabelBar v-bind="labelBarProps('bottom', 'homePower')">
					<shopicon-regular-home></shopicon-regular-home>
				</LabelBar>
				<LabelBar
					v-for="(lp, index) in loadpoints"
					:key="index"
					v-bind="labelBarProps('bottom', 'loadpoints', lp.chargePower)"
				>
					<VehicleIcon :names="[lp.icon]" />
				</LabelBar>
				<LabelBar v-bind="labelBarProps('bottom', 'batteryCharge')">
					<BatteryIcon :soc="batterySoc" :gridCharge="batteryGridCharge" />
				</LabelBar>
				<LabelBar v-bind="labelBarProps('bottom', 'gridExport')">
					<shopicon-regular-powersupply></shopicon-regular-powersupply>
				</LabelBar>
				<LabelBar v-bind="labelBarProps('bottom', 'unknownOutput')">
					<QuestionIcon />
				</LabelBar>
			</div>
			<div class="label-scale-name">Out</div>
		</div>
		<BatteryIcon hold class="battery-hold" :class="{ 'battery-hold--active': batteryHold }" />
	</div>
</template>

<script lang="ts">
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import BatteryIcon from "./BatteryIcon.vue";
import LabelBar from "./LabelBar.vue";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import VehicleIcon from "../VehicleIcon";
import QuestionIcon from "../MaterialIcon/Question.vue";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/home";
import { defineComponent, type PropType } from "vue";
import type { UiLoadpoint } from "@/types/evcc";

export default defineComponent({
	name: "Visualization",
	components: { BatteryIcon, LabelBar, AnimatedNumber, VehicleIcon, QuestionIcon },
	mixins: [formatter],
	props: {
		gridImport: { type: Number, default: 0 },
		selfPv: { type: Number, default: 0 },
		selfBattery: { type: Number, default: 0 },
		pvExport: { type: Number, default: 0 },
		loadpoints: { type: Array as PropType<UiLoadpoint[]>, default: () => [] },
		batteryCharge: { type: Number, default: 0 },
		batteryDischarge: { type: Number, default: 0 },
		batteryHold: { type: Boolean, default: false },
		batteryGridCharge: { type: Boolean, default: false },
		pvProduction: { type: Number, default: 0 },
		homePower: { type: Number, default: 0 },
		batterySoc: { type: Number, default: 0 },
		powerUnit: { type: String as PropType<POWER_UNIT>, default: POWER_UNIT.KW },
		inPower: { type: Number, default: 0 },
		outPower: { type: Number, default: 0 },
	},
	data() {
		return { width: 0 };
	},
	computed: {
		gridExport() {
			return this.applyThreshold(this.pvExport);
		},
		totalRaw() {
			return this.gridImport + this.selfPv + this.selfBattery + this.pvExport;
		},
		gridImportAdjusted() {
			return this.applyThreshold(this.gridImport);
		},
		selfPvAdjusted() {
			return this.applyThreshold(this.selfPv);
		},
		selfBatteryAdjusted() {
			return this.applyThreshold(this.selfBattery);
		},
		pvExportAdjusted() {
			return this.applyThreshold(this.pvExport);
		},
		totalAdjusted() {
			return (
				this.gridImportAdjusted +
				this.selfPvAdjusted +
				this.selfBatteryAdjusted +
				this.pvExportAdjusted
			);
		},
		unknownImport() {
			// input/output mismatch > 10%
			return this.applyThreshold(Math.max(0, this.outPower - this.inPower), 10);
		},
		unknownOutput() {
			// input/output mismatch > 10%
			return this.applyThreshold(Math.max(0, this.inPower - this.outPower), 10);
		},
		unknownPower() {
			if (this.unknownImport || this.unknownOutput) {
				const total = Math.max(this.inPower, this.outPower);
				return Math.abs(total - this.totalAdjusted);
			}
			return 0;
		},
		visualizationReady() {
			return this.totalAdjusted > 0 && this.width > 0;
		},
	},

	mounted() {
		this.$nextTick(function () {
			window.addEventListener("resize", this.updateElementWidth);
			this.updateElementWidth();
		});
	},
	beforeUnmount() {
		window.removeEventListener("resize", this.updateElementWidth);
	},
	methods: {
		widthTotal(power: number) {
			if (this.totalAdjusted === 0 || power === 0) return "0";
			return (100 / this.totalAdjusted) * power + "%";
		},
		fmtBarValue(watt: number) {
			if (!this.enoughSpaceForValue(watt)) {
				return "";
			}
			const withUnit = this.enoughSpaceForUnit(watt);
			return this.fmtW(watt, this.powerUnit, withUnit);
		},
		powerLabelAvailableSpace(power: number) {
			if (this.totalAdjusted === 0) return 0;
			const percent = (100 / this.totalAdjusted) * power;
			return (this.width / 100) * percent;
		},
		enoughSpaceForValue(power: number) {
			return this.powerLabelAvailableSpace(power) > 40;
		},
		enoughSpaceForUnit(power: number) {
			return this.powerLabelAvailableSpace(power) > 60;
		},
		hideLabelIcon(power: number, minWidth = 32) {
			if (this.totalAdjusted === 0) return true;
			const percent = (100 / this.totalAdjusted) * power;
			return (this.width / 100) * percent < minWidth;
		},
		applyThreshold(power: number, threshold = 2) {
			const percent = (100 / this.totalRaw) * power;
			return percent < threshold ? 0 : power;
		},
		updateElementWidth() {
			this.width = this.$refs["site_progress"]?.getBoundingClientRect().width ?? 0;
		},
		labelBarProps(position: string, name: string, val?: number) {
			const value = val === undefined ? (this as any)[name] : val;
			const minWidth = 40;
			return {
				value,
				hideIcon: this.hideLabelIcon(value, minWidth),
				style: { "flex-basis": this.widthTotal(value) },
				[position]: true,
			};
		},
	},
});
</script>
<style scoped>
.site-progress {
	--height: 2.5rem;
	height: var(--height);
	border-radius: 10px;
	display: flex;
	overflow: hidden;
	margin-right: 1.2rem;
}
.label-scale-name {
	color: var(--evcc-gray);
	flex-basis: 1.2rem;
	flex-grow: 0;
	flex-shrink: 0;
	writing-mode: tb-rl;
	line-height: 1;
	text-align: center;
}
.site-progress-bar {
	display: flex;
	justify-content: center;
	align-items: center;
	overflow: hidden;
	position: relative;
	width: 0;
}
.visualization--ready .site-progress-bar {
	transition-property: width;
	transition-duration: var(--evcc-transition-medium);
	transition-timing-function: linear;
}
.grid-import {
	background-color: var(--evcc-grid);
	color: var(--bs-white);
}
html.dark .grid-import {
	color: var(--bs-dark);
}

.self-pv {
	background-color: var(--evcc-pv);
	color: var(--bs-dark);
}
.self-battery {
	background-color: var(--evcc-battery);
	color: var(--bs-dark);
}
.pv-export {
	background-color: var(--evcc-export);
	color: var(--bs-dark);
}
.unknown-power {
	background-color: var(--evcc-gray);
	color: var(--bs-dark);
}
.power {
	display: block;
	margin: 0 0.2rem;
	white-space: nowrap;
	overflow: hidden;
}
.visualization--ready :deep(.label-bar) {
	transition-property: flex-basis, opacity;
	transition-duration: var(--evcc-transition-medium), var(--evcc-transition-fast);
	transition-timing-function: linear, ease;
}
.visualization--ready :deep(.label-bar-icon) {
	transition-duration: var(--evcc-transition-very-fast), 500ms;
}
.battery-hold {
	position: absolute;
	top: 2.5rem;
	right: -0.25rem;
	color: var(--evcc-gray);
	opacity: 0;
	transition-property: opacity;
	transition-duration: var(--evcc-transition-medium);
	transition-timing-function: linear;
}
.battery-hold--active {
	opacity: 1;
}
</style>
