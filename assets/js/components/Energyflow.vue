<template>
	<div class="row align-items-start align-items-md-center mt-4" @click="toggleDetails">
		<div class="d-none d-md-flex col-12 col-md-3 col-lg-4">
			<h4>
				<span class="h6">{{ $t("main.energyflow.titleSup") }}</span>
				<br />
				{{ $t("main.energyflow.title") }}
			</h4>
		</div>
		<div
			class="col-12 flex-grow-1"
			:class="`col-md-${showDetails ? '6' : '8'} col-lg-${showDetails ? '5' : '7'}`"
		>
			<div class="label-scale">
				<div class="d-flex justify-content-end">
					<div
						class="label-bar label-bar--down"
						v-if="batteryDischarge"
						:style="{ width: widthTotal(batteryDischarge) }"
					>
						<div class="label-bar-scale">
							<div class="label-bar-icon">
								<fa-icon :icon="batteryIcon"></fa-icon>
								<fa-icon icon="caret-right"></fa-icon>
							</div>
						</div>
					</div>
					<div class="label-bar label-bar--down" :style="{ width: widthTotal(pvPower) }">
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
					:style="{ width: widthTotal(gridImport) }"
				>
					<span class="power" :class="{ 'd-none': hidePowerLabel(gridImport) }">
						{{ kw(gridImport) }}
					</span>
				</div>
				<div
					class="site-progress-bar self-consumption"
					:style="{ width: widthTotal(selfConsumption) }"
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
				<div class="site-progress-bar pv-export" :style="{ width: widthTotal(pvExport) }">
					<span class="power" :class="{ 'd-none': hidePowerLabel(pvExport) }">
						{{ kw(pvExport) }}
					</span>
				</div>
			</div>
			<div class="label-scale">
				<div class="d-flex justify-content-start">
					<div
						class="label-bar label-bar--up"
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
						:style="{ width: widthTotal(batteryCharge) }"
					>
						<div class="label-bar-scale">
							<div class="label-bar-icon">
								<fa-icon :icon="batteryIcon"></fa-icon>
								<fa-icon icon="caret-left"></fa-icon>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div class="col-12 col-sm-6 col-md-3 col-lg-3 mt-2 mt-md-0 small" v-if="showDetails">
			<div class="d-flex justify-content-between" data-test-pv-production>
				<span class="details-icon"><fa-icon icon="sun"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{
					$t("main.energyflow.pvProduction")
				}}</span>
				<span class="text-end text-nowrap ps-1">{{ kw(pvPower) }}</span>
			</div>
			<div class="d-flex justify-content-between" data-test-house-consumption>
				<span class="details-icon"><fa-icon icon="home"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{
					$t("main.energyflow.houseConsumption")
				}}</span>
				<span class="text-end text-nowrap ps-1">{{ kw(houseConsumption) }}</span>
			</div>
			<div v-if="batteryConfigured" class="d-flex justify-content-between" data-test-battery>
				<span class="details-icon">
					<fa-icon :icon="batteryIcon"></fa-icon>
					<fa-icon icon="caret-left" v-if="batteryCharge"></fa-icon>
					<fa-icon icon="caret-right" v-if="batteryDischarge"></fa-icon>
				</span>
				<span class="text-nowrap flex-grow-1 text-truncate">
					<span v-if="batteryCharge">{{ $t("main.energyflow.batteryCharge") }}</span>
					<span v-else-if="batteryDischarge">{{
						$t("main.energyflow.batteryDischarge")
					}}</span>
					<span v-else>{{ $t("main.energyflow.battery") }}</span>
				</span>
				<span class="text-end text-nowrap ps-1">
					({{ batterySoC }}%)
					{{ kw(Math.abs(batteryPower)) }}
				</span>
			</div>
		</div>
		<div
			class="
				col-12 col-sm-6
				offset-md-3
				col-md-6
				offset-lg-4
				col-lg-5
				d-block d-md-flex
				justify-content-between
				mt-2
				small
			"
			v-if="showDetails"
		>
			<div class="text-nowrap d-flex d-md-block" data-test-grid-import>
				<span class="color-grid details-icon"><fa-icon icon="square"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{ $t("main.energyflow.gridImport") }}</span>
				<span class="text-end text-nowrap d-md-none">{{ kw(gridImport) }}</span>
			</div>
			<div class="text-nowrap d-flex d-md-block" data-test-self-consumption>
				<span class="color-self details-icon"><fa-icon icon="square"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{
					$t("main.energyflow.selfConsumption")
				}}</span>
				<span class="text-end text-nowrap d-md-none">{{ kw(selfConsumption) }}</span>
			</div>
			<div class="text-nowrap d-flex d-md-block" data-test-pv-export>
				<span class="color-export details-icon"><fa-icon icon="square"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{ $t("main.energyflow.pvExport") }}</span>
				<span class="text-end text-nowrap d-md-none">{{ kw(pvExport) }}</span>
			</div>
		</div>
	</div>
</template>

<script>
import "../icons";
import formatter from "../mixins/formatter";

export default {
	name: "energyflow",
	props: {
		gridConfigured: Boolean,
		gridPower: { type: Number, default: 0 },
		pvConfigured: Boolean,
		pvPower: { type: Number, default: 0 },
		batteryConfigured: Boolean,
		batteryPower: { type: Number, default: 0 },
		batterySoC: { type: Number, default: 0 },
	},
	data: function () {
		return { showDetails: false, width: 0 };
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
		rawTotal: function () {
			return Math.max(0, this.batteryPower) + Math.max(0, this.gridPower) + this.pvPower;
		},
		gridImport: function () {
			return Math.max(0, this.gridPower);
		},
		pvConsumption: function () {
			return this.applyThreshold(
				Math.min(this.pvPower, this.pvPower + this.gridPower - this.batteryCharge)
			);
		},
		batteryDischarge: function () {
			return this.applyThreshold(Math.max(0, this.batteryPower));
		},
		batteryCharge: function () {
			return this.applyThreshold(Math.min(0, this.batteryPower) * -1);
		},
		houseConsumption: function () {
			return this.gridImport + this.pvConsumption + this.batteryDischarge;
		},
		selfConsumption: function () {
			return this.batteryDischarge + this.pvConsumption + this.batteryCharge;
		},
		pvExport: function () {
			return this.applyThreshold(Math.min(0, this.gridPower) * -1);
		},
		total: function () {
			return this.gridImport + this.selfConsumption + this.pvExport;
		},
		batteryIcon: function () {
			if (this.batterySoC > 80) return "battery-full";
			if (this.batterySoC > 60) return "battery-three-quarters";
			if (this.batterySoC > 40) return "battery-half";
			if (this.batterySoC > 20) return "battery-quarter";
			return "battery-empty";
		},
	},
	methods: {
		widthTotal: function (power) {
			return (100 / this.total) * power + "%";
		},
		kw: function (watt) {
			return Math.max(0, watt / 1000).toFixed(1) + " kW";
		},
		hidePowerLabel(power) {
			const minWidth = 75;
			const percent = (100 / this.total) * power;
			return (this.width / 100) * percent < minWidth;
		},
		applyThreshold(power) {
			// set value to 0 if it doesn't exceed 200kW
			return power < 200 ? 0 : power;
		},
		toggleDetails() {
			this.showDetails = !this.showDetails;
			this.$nextTick(() => this.updateElementWidth());
		},
		updateElementWidth() {
			this.width = this.$refs.site_progress.getBoundingClientRect().width;
		},
	},
};
</script>
<style scoped>
.site-progress {
	margin: 0.25rem 0;
	border-radius: 5px;
	display: flex;
	overflow: hidden;
	cursor: pointer;
}
.site-progress-bar {
	display: flex;
	transition-property: width;
	transition-duration: 500ms;
	transition-timing-function: linear;
	justify-content: center;
	align-items: center;
	overflow: hidden;
	height: 1.5rem;
	position: relative;
}
.site-progress-bar::before,
.site-progress-bar::after {
	content: "";
	top: 0;
	bottom: 0;
	width: 1px;
	background: white;
	position: absolute;
}
.site-progress-bar::before {
	left: 0px;
}
.site-progress-bar::after {
	right: 0px;
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
.color-grid {
	color: var(--evcc-grid);
}
.color-self {
	color: var(--evcc-self);
}
.color-export {
	color: var(--evcc-export);
}
.details-icon {
	text-align: center;
	width: 30px;
	margin-right: 0.25rem;
	white-space: nowrap;
	flex-shrink: 0;
}
.power {
	display: block;
	margin: 0 0.5rem;
	white-space: nowrap;
}
.label-bar {
	margin: 0;
	height: 1.5rem;
	padding: 0.5rem 0;
	transition-property: width;
	transition-duration: 500ms;
	transition-timing-function: linear;
	cursor: pointer;
	overflow: hidden;
}
.label-bar-scale {
	border: 1px solid var(--bs-gray);
	height: 7px;
	background: none;
	display: flex;
	justify-content: center;
	align-items: center;
	white-space: nowrap;
}
.label-bar--down .label-bar-scale {
	border-bottom: 5px solid transparent;
}
.label-bar--up .label-bar-scale {
	border-top: 5px solid transparent;
}
.label-bar-icon {
	background-color: white;
	color: var(--bs-gray);
	padding: 0 0.75rem;
}
</style>
