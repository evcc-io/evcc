<template>
	<div
		class="row align-items-start align-items-md-center mt-4"
		:class="{ 'show-details': showDetails }"
	>
		<div class="d-none d-md-flex col-12 col-md-3 col-lg-4">
			<h4><span class="h6">Aktuelle</span><br />Energiebilanz</h4>
		</div>
		<div
			class="col-12 flex-grow-1"
			:class="`col-md-${showDetails ? '6' : '8'} col-lg-${showDetails ? '5' : '7'}`"
			@click="toggleDetails"
		>
			<div class="label-scale">
				<div class="d-flex justify-content-start">
					<div class="label-bar label-bar--down" :style="{ width: widthTotal(usage) }">
						<div class="label-bar-scale">
							<div class="label-bar-icon">
								<fa-icon icon="home"></fa-icon>
							</div>
						</div>
					</div>
					<div
						class="label-bar label-bar--down"
						v-if="batteryCharge"
						:style="{ width: widthTotal(batteryCharge) }"
					>
						<div class="label-bar-scale">
							<div class="label-bar-icon">
								<fa-icon :icon="batteryIcon"></fa-icon> ↑
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="site-progress" ref="site_progress">
				<div class="site-progress-bar grid-usage" :style="{ width: widthTotal(gridUsage) }">
					<span class="power" :class="{ 'd-none': hidePowerLabel(gridUsage) }">
						{{ kw(gridUsage) }}
					</span>
				</div>
				<div
					class="site-progress-bar self-usage"
					:style="{ width: widthTotal(batteryUsage + pvUsage + batteryCharge) }"
				>
					<span
						class="power"
						:class="{
							'd-none': hidePowerLabel(batteryUsage + pvUsage + batteryCharge),
						}"
					>
						{{ kw(batteryUsage + pvUsage + batteryCharge) }}
					</span>
				</div>
				<div class="site-progress-bar surplus" :style="{ width: widthTotal(pvExport) }">
					<span class="power" :class="{ 'd-none': hidePowerLabel(pvExport) }">
						{{ kw(pvExport) }}
					</span>
				</div>
			</div>
			<div class="label-scale">
				<div class="d-flex justify-content-end">
					<div
						class="label-bar label-bar--up"
						v-if="batteryUsage"
						:style="{ width: widthTotal(batteryUsage) }"
					>
						<div class="label-bar-scale label-bar-scale--up">
							<div class="label-bar-icon">
								<fa-icon :icon="batteryIcon"></fa-icon> ↓
							</div>
						</div>
					</div>
					<div class="label-bar label-bar--up" :style="{ width: widthTotal(pvPower) }">
						<div class="label-bar-scale label-bar-scale--up">
							<div class="label-bar-icon">
								<fa-icon icon="sun"></fa-icon>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div
			class="col-12 col-sm-6 col-md-3 col-lg-3 mt-2 mt-md-0 small"
			@click="toggleDetails"
			v-if="showDetails"
		>
			<div class="d-flex justify-content-between">
				<span class="details-icon"><fa-icon icon="home"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{
					$t("main.siteVisualization.consumption")
				}}</span>
				<span class="text-end text-nowrap ps-1">{{ kw(usage) }}</span>
			</div>
			<div class="d-flex justify-content-between">
				<span class="details-icon"><fa-icon icon="sun"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{ $t("main.siteVisualization.pv") }}</span>
				<span class="text-end text-nowrap ps-1">{{ kw(pvPower) }}</span>
			</div>
			<div v-if="batteryConfigured" class="d-flex justify-content-between">
				<span class="details-icon"><fa-icon :icon="batteryIcon"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">
					{{ $t("main.siteVisualization.battery") }}
					<span v-if="batteryCharge"> ↑ </span>
					<span v-if="batteryUsage"> ↓ </span>
					{{ batterySoC }}%
				</span>
				<span class="text-end text-nowrap ps-1">
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
			@click="toggleDetails"
			v-if="showDetails"
		>
			<div class="text-nowrap d-flex d-md-block">
				<span class="color-grid details-icon"><fa-icon icon="square"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{ $t("main.siteVisualization.grid") }}</span>
				<span class="text-end text-nowrap d-md-none">{{ kw(gridUsage) }}</span>
			</div>
			<div class="text-nowrap d-flex d-md-block">
				<span class="color-self details-icon"><fa-icon icon="square"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{ $t("main.siteVisualization.self") }}</span>
				<span class="text-end text-nowrap d-md-none">{{ kw(selfUsage) }}</span>
			</div>
			<div class="text-nowrap d-flex d-md-block">
				<span class="color-export details-icon"><fa-icon icon="square"></fa-icon></span>
				<span class="text-nowrap flex-grow-1">{{
					$t("main.siteVisualization.export")
				}}</span>
				<span class="text-end text-nowrap d-md-none">{{ kw(pvExport) }}</span>
			</div>
		</div>
	</div>
</template>

<script>
import "../icons";
import formatter from "../mixins/formatter";

export default {
	name: "SiteVisualization",
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
		gridUsage: function () {
			return Math.max(0, this.gridPower);
		},
		pvUsage: function () {
			return this.applyThreshold(
				Math.min(this.pvPower, this.pvPower + this.gridPower - this.batteryCharge)
			);
		},
		batteryUsage: function () {
			return this.applyThreshold(Math.max(0, this.batteryPower));
		},
		usage: function () {
			return this.gridUsage + this.pvUsage + this.batteryUsage;
		},
		pvExport: function () {
			return this.applyThreshold(Math.min(0, this.gridPower) * -1);
		},
		batteryCharge: function () {
			return this.applyThreshold(Math.min(0, this.batteryPower) * -1);
		},
		selfUsage: function () {
			return this.pvUsage + this.batteryUsage;
		},
		surplus: function () {
			return this.pvExport + this.batteryCharge;
		},
		total: function () {
			return this.usage + this.surplus;
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
		widthSurplus: function (power) {
			return (100 / this.surplus) * power + "%";
		},
		widthUsage: function (power) {
			return (100 / this.usage) * power + "%";
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
.grid-usage {
	background-color: var(--evcc-grid);
	color: var(--bs-light);
}
.self-usage {
	background-color: var(--evcc-self);
	color: var(--bs-light);
}
.surplus {
	background-color: var(--evcc-surplus);
	color: var(--bs-dark);
}
.color-grid {
	color: var(--evcc-grid);
}
.color-self {
	color: var(--evcc-self);
}
.color-export {
	color: var(--evcc-surplus);
}
.details-icon {
	text-align: center;
	width: 2.5rem;
}

.power {
	display: block;
	margin: 0 0.5rem;
	white-space: nowrap;
	opacity: 0;
	transition: opacity 100ms ease;
}
.show-details .power {
	opacity: 1;
}

.usage-label,
.production-label {
	flex-basis: 33%;
}
.battery-label {
	position: relative;
}
.usage-label {
	justify-content: flex-start;
}
.production-label {
	justify-content: flex-end;
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
.table > tbody > tr:last-child > td {
	border-bottom: none;
}
</style>
