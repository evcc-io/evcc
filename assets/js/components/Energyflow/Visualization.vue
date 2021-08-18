<template>
	<div class="visualization" :class="{ 'visualization--ready': visualizationReady }">
		<div class="label-scale">
			<div class="d-flex justify-content-end">
				<div
					class="label-bar label-bar--down"
					v-if="batteryDischarge"
					:class="{ 'label-bar--hide-icon': hideLabelIcon(batteryDischarge, 44) }"
					:style="{ width: widthTotal(batteryDischarge) }"
				>
					<div class="label-bar-scale">
						<div class="label-bar-icon">
							<BatteryIcon :soc="batterySoC" discharge />
						</div>
					</div>
				</div>
				<div
					class="label-bar label-bar--down"
					v-if="selfConsumptionAdjusted || pvExportAdjusted"
					:class="{ 'label-bar--hide-icon': hideLabelIcon(pvProduction) }"
					:style="{ width: widthTotal(pvProduction) }"
				>
					<div class="label-bar-scale">
						<div class="label-bar-icon">
							<fa-icon icon="sun"></fa-icon>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div class="site-progress" ref="site_progress">
			<div
				class="site-progress-bar grid-import"
				:style="{ width: widthTotal(gridImportAdjusted) }"
			>
				<span class="power" v-if="powerLabelEnoughSpace(gridImport)">
					{{ kw(gridImport) }}
				</span>
				<span class="power" v-else-if="powerLabelSomeSpace(gridImport)">
					{{ kwNoUnit(gridImport) }}
				</span>
			</div>
			<div
				class="site-progress-bar self-consumption"
				:style="{ width: widthTotal(selfConsumptionAdjusted) }"
			>
				<span class="power" v-if="powerLabelEnoughSpace(selfConsumption)">
					{{ kw(selfConsumption) }}
				</span>
				<span class="power" v-else-if="powerLabelSomeSpace(selfConsumption)">
					{{ kwNoUnit(selfConsumption) }}
				</span>
			</div>
			<div
				class="site-progress-bar pv-export"
				:style="{ width: widthTotal(pvExportAdjusted) }"
			>
				<span class="power" v-if="powerLabelEnoughSpace(pvExport)">
					{{ kw(pvExport) }}
				</span>
				<span class="power" v-else-if="powerLabelSomeSpace(pvExport)">
					{{ kwNoUnit(pvExport) }}
				</span>
			</div>
			<div class="site-progress-bar bg-light border no-wrap w-100" v-if="totalAdjusted <= 0">
				<span>{{ $t("main.energyflow.noEnergy") }}</span>
			</div>
		</div>
		<div class="label-scale">
			<div class="d-flex justify-content-start">
				<div
					class="label-bar label-bar--up"
					v-if="gridImportAdjusted || selfConsumptionAdjusted"
					:class="{ 'label-bar--hide-icon': hideLabelIcon(houseConsumption) }"
					:style="{ width: widthTotal(houseConsumption) }"
				>
					<div class="label-bar-scale">
						<div class="label-bar-icon">
							<fa-icon icon="home"></fa-icon>
						</div>
					</div>
				</div>
				<div
					class="label-bar label-bar--up"
					v-if="batteryCharge"
					:class="{ 'label-bar--hide-icon': hideLabelIcon(batteryCharge, 44) }"
					:style="{ width: widthTotal(batteryCharge) }"
				>
					<div class="label-bar-scale">
						<div class="label-bar-icon">
							<BatteryIcon :soc="batterySoC" charge />
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import "../../icons";
import formatter from "../../mixins/formatter";
import BatteryIcon from "./BatteryIcon.vue";

export default {
	name: "Visualization",
	components: { BatteryIcon },
	props: {
		showDetails: Boolean,
		gridImport: { type: Number, default: 0 },
		selfConsumption: { type: Number, default: 0 },
		pvExport: { type: Number, default: 0 },
		batteryCharge: { type: Number, default: 0 },
		batteryDischarge: { type: Number, default: 0 },
		pvProduction: { type: Number, default: 0 },
		houseConsumption: { type: Number, default: 0 },
		batterySoC: { type: Number, default: 0 },
		valuesInKw: { type: Boolean, default: false },
	},
	data: function () {
		return { width: 0, visualizationReady: false };
	},
	mounted: function () {
		this.$nextTick(function () {
			window.addEventListener("resize", this.updateElementWidth);
			this.updateElementWidth();
		});
	},
	beforeDestroy() {
		window.removeEventListener("resize", this.updateElementWidth);
	},
	mixins: [formatter],
	computed: {
		totalRaw: function () {
			return this.gridImport + this.selfConsumption + this.pvExport;
		},
		gridImportAdjusted: function () {
			return this.applyThreshold(this.gridImport);
		},
		selfConsumptionAdjusted: function () {
			return this.applyThreshold(this.selfConsumption);
		},
		pvExportAdjusted: function () {
			return this.applyThreshold(this.pvExport);
		},
		totalAdjusted: function () {
			return this.gridImportAdjusted + this.selfConsumptionAdjusted + this.pvExportAdjusted;
		},
	},
	watch: {
		showDetails: function () {
			this.$nextTick(() => this.updateElementWidth());
		},
		totalAdjusted: function () {
			if (!this.visualizationReady && this.totalAdjusted > 0)
				setTimeout(() => {
					this.visualizationReady = true;
				}, 500);
		},
	},
	methods: {
		widthTotal: function (power) {
			if (this.totalAdjusted === 0) return "0%";
			return (100 / this.totalAdjusted) * power + "%";
		},
		kw: function (watt) {
			return this.fmtKw(watt, this.valuesInKw, true);
		},
		kwNoUnit: function (watt) {
			return this.fmtKw(watt, this.valuesInKw, false);
		},
		powerLabelAvailableSpace(power) {
			if (this.totalAdjusted === 0) return 0;
			const percent = (100 / this.totalAdjusted) * power;
			return (this.width / 100) * percent;
		},
		powerLabelEnoughSpace(power) {
			return this.powerLabelAvailableSpace(power) > 60;
		},
		powerLabelSomeSpace(power) {
			return this.powerLabelAvailableSpace(power) > 35;
		},
		hideLabelIcon(power, minWidth = 32) {
			if (this.totalAdjusted === 0) return true;
			const percent = (100 / this.totalAdjusted) * power;
			return (this.width / 100) * percent < minWidth;
		},
		applyThreshold(power) {
			const percent = (100 / this.totalRaw) * power;
			return percent < 2 ? 0 : power;
		},
		updateElementWidth() {
			this.width = this.$refs.site_progress.getBoundingClientRect().width;
		},
	},
};
</script>
<style scoped>
.site-progress {
	--height: 38px;
	height: var(--height);
	margin: 0.25rem 0;
	border-radius: 5px;
	display: flex;
	overflow: hidden;
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
	transition-duration: 500ms;
	transition-timing-function: linear;
}
.grid-import {
	background-color: var(--evcc-grid);
	color: var(--bs-white);
}
.self-consumption {
	background-color: var(--evcc-self);
	color: var(--bs-white);
}
.pv-export {
	background-color: var(--evcc-export);
	color: var(--bs-dark);
}
.power {
	display: block;
	margin: 0 0.2rem;
	white-space: nowrap;
	overflow: hidden;
}
.label-bar {
	width: 0;
	margin: 0;
	height: 1.7rem;
	padding: 0.6rem 0;
	opacity: 1;
	overflow: hidden;
}
.visualization--ready .label-bar {
	transition-property: width, opacity;
	transition-duration: 500ms, 250ms;
	transition-timing-function: linear, ease;
}
.label-bar--down:first-child {
	margin-right: -1px;
}
.label-bar--up:last-child {
	margin-left: -1px;
}
.label-bar-scale {
	border: 1px solid var(--bs-gray);
	height: 0.5rem;
	background: none;
	display: flex;
	justify-content: center;
	align-items: center;
	white-space: nowrap;
}
.label-bar--down .label-bar-scale {
	border-bottom: none;
}
.label-bar--up .label-bar-scale {
	border-top: none;
}
.label-bar--down:first-child .label-bar-scale {
	border-start-start-radius: 4px;
}
.label-bar--down:last-child .label-bar-scale {
	border-start-end-radius: 4px;
}
.label-bar--up:first-child .label-bar-scale {
	border-end-start-radius: 4px;
}
.label-bar--up:last-child .label-bar-scale {
	border-end-end-radius: 4px;
}
.label-bar-icon {
	background-color: white;
	color: var(--bs-gray);
	padding: 0 0.3rem;
	opacity: 1;
}
.visualization--ready .label-bar-icon {
	transition: opacity 250ms ease-in;
}
.label-bar--down .label-bar-icon {
	margin-top: -6px;
}
.label-bar--up .label-bar-icon {
	margin-top: 6px;
}
.label-bar--hide-icon .label-bar-icon {
	opacity: 0;
}
.label-bar--hidden {
	opacity: 0;
}
</style>
