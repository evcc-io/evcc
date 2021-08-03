<template>
	<div>
		<div class="label-scale">
			<div class="d-flex justify-content-end">
				<div
					class="label-bar label-bar--down"
					:class="{
						'label-bar--hide-icon': hideLabelIcon(batteryDischarge),
						'label-bar--hidden': !selfConsumptionAdjusted,
					}"
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
					:class="{
						'label-bar--hide-icon': hideLabelIcon(pvProduction),
						'label-bar--hidden': !selfConsumptionAdjusted && !pvExportAdjusted,
					}"
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
				<span class="power" :class="{ 'd-none': hidePowerLabel(gridImport) }">
					{{ kw(gridImport) }}
				</span>
			</div>
			<div
				class="site-progress-bar self-consumption"
				:style="{ width: widthTotal(selfConsumptionAdjusted) }"
			>
				<span
					class="power"
					:class="{
						'd-none': hidePowerLabel(selfConsumption),
					}"
				>
					{{ kw(selfConsumption) }}
				</span>
			</div>
			<div
				class="site-progress-bar pv-export"
				:style="{ width: widthTotal(pvExportAdjusted) }"
			>
				<span class="power" :class="{ 'd-none': hidePowerLabel(pvExport) }">
					{{ kw(pvExport) }}
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
					:class="{
						'label-bar--hide-icon': hideLabelIcon(houseConsumption),
						'label-bar--hidden': !gridImportAdjusted && !selfConsumptionAdjusted,
					}"
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
					:class="{
						'label-bar--hide-icon': hideLabelIcon(batteryCharge),
						'label-bar--hidden': !selfConsumptionAdjusted,
					}"
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
	},
	data: function () {
		return { width: 0 };
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
	},
	methods: {
		widthTotal: function (power) {
			if (this.totalAdjusted === 0) return "0%";
			return (100 / this.totalAdjusted) * power + "%";
		},
		kw: function (watt) {
			return Math.max(0, watt / 1000).toFixed(1) + " kW";
		},
		hidePowerLabel(power) {
			if (this.totalAdjusted === 0) return true;
			const minWidth = 75;
			const percent = (100 / this.totalAdjusted) * power;
			return (this.width / 100) * percent < minWidth;
		},
		hideLabelIcon(power) {
			if (this.totalAdjusted === 0) return true;
			const minWidth = 60;
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
	transition-property: width;
	transition-duration: 500ms;
	transition-timing-function: linear;
	justify-content: center;
	align-items: center;
	overflow: hidden;
	position: relative;
	width: 0;
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
	margin: 0 0.5rem;
	white-space: nowrap;
}
.label-bar {
	width: 0;
	margin: 0;
	height: 1.7rem;
	padding: 0.6rem 0;
	opacity: 1;
	transition-property: width, opacity;
	transition-duration: 500ms, 250ms;
	transition-timing-function: linear, ease;
	overflow: hidden;
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
.label-bar-icon {
	background-color: white;
	color: var(--bs-gray);
	padding: 0 0.75rem;
	opacity: 1;
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
