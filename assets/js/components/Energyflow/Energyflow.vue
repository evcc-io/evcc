<template>
	<div class="row energyflow pb-4">
		<Visualization
			class="col-12"
			:gridImport="gridImport"
			:selfConsumption="selfConsumption"
			:loadpoints="loadpointsPower"
			:pvExport="pvExport"
			:batteryCharge="batteryCharge"
			:batteryDischarge="batteryDischarge"
			:pvProduction="pvProduction"
			:homePower="homePower"
			:batterySoC="batterySoC"
			:valuesInKw="valuesInKw"
		/>
		<div class="row mb-4">
			<div class="col-12 d-flex justify-content-around">
				<div class="d-flex flex-nowrap">
					<span class="color-self details-icon"><fa-icon icon="square"></fa-icon></span>
					<span>{{ $t("main.energyflow.selfConsumption") }}</span>
				</div>
				<div v-if="gridImport > 0" class="d-flex flex-nowrap">
					<span class="color-grid details-icon"><fa-icon icon="square"></fa-icon></span>
					<span>{{ $t("main.energyflow.gridImport") }}</span>
				</div>
				<div v-else class="d-flex flex-nowrap">
					<span class="color-export details-icon"><fa-icon icon="square"></fa-icon></span>
					<span>{{ $t("main.energyflow.pvExport") }}</span>
				</div>
			</div>
		</div>
		<div class="row">
			<div class="col-6">
				<h4>In</h4>
				<div class="d-flex justify-content-between" data-test-pv-production>
					<span class="details-icon text-muted"
						><shopicon-regular-sun></shopicon-regular-sun
					></span>
					<span class="text-nowrap flex-grow-1">{{
						$t("main.energyflow.pvProduction")
					}}</span>
					<span class="text-end text-nowrap ps-1">{{ kw(pvProduction) }}</span>
				</div>
				<div class="d-flex justify-content-between">
					<span class="details-icon text-muted">
						<GridIcon :import="gridImport > 0" />
					</span>
					<span class="text-nowrap flex-grow-1 text-truncate">
						<span>{{ $t("main.energyflow.gridImport") }}</span>
					</span>
					<span class="text-end text-nowrap ps-1">
						{{ kw(gridImport) }}
					</span>
				</div>
				<div
					v-if="batteryConfigured"
					class="d-flex justify-content-between"
					data-test-battery
				>
					<span class="details-icon text-muted">
						<BatteryIcon :soc="batterySoC" :discharge="batteryDischarge > 0" />
					</span>
					<span class="text-nowrap flex-grow-1 text-truncate">
						<span>{{ $t("main.energyflow.batteryDischarge") }}</span>
					</span>
					<span class="text-end text-nowrap ps-1">
						{{ batterySoC }}% /
						{{ kw(Math.abs(batteryDischarge)) }}
					</span>
				</div>
			</div>
			<div class="col-6">
				<h4>Out</h4>
				<div class="d-flex justify-content-between" data-test-home-power>
					<span class="details-icon text-muted"
						><shopicon-regular-home></shopicon-regular-home
					></span>
					<span class="text-nowrap flex-grow-1">{{
						$t("main.energyflow.homePower")
					}}</span>
					<span class="text-end text-nowrap ps-1">{{ kw(homePower) }}</span>
				</div>
				<div class="d-flex justify-content-between" data-test-loadpoints>
					<span class="details-icon text-muted"
						><shopicon-regular-car3></shopicon-regular-car3
					></span>
					<span class="text-nowrap flex-grow-1">{{
						$tc("main.energyflow.loadpoints", activeLoadpointsCount, {
							count: activeLoadpointsCount,
						})
					}}</span>
					<span class="text-end text-nowrap ps-1">{{ kw(loadpointsPower) }}</span>
				</div>
				<div class="d-flex justify-content-between">
					<span class="details-icon text-muted">
						<GridIcon :export="pvExport > 0" />
					</span>
					<span class="text-nowrap flex-grow-1 text-truncate">
						<span>{{ $t("main.energyflow.pvExport") }}</span>
					</span>
					<span class="text-end text-nowrap ps-1">
						{{ kw(pvExport) }}
					</span>
				</div>
				<div
					v-if="batteryConfigured"
					class="d-flex justify-content-between"
					data-test-battery
				>
					<span class="details-icon text-muted">
						<BatteryIcon :soc="batterySoC" :charge="batteryCharge > 0" />
					</span>
					<span class="text-nowrap flex-grow-1 text-truncate">
						<span>{{ $t("main.energyflow.batteryCharge") }}</span>
					</span>
					<span class="text-end text-nowrap ps-1">
						{{ batterySoC }}% /
						{{ kw(Math.abs(batteryCharge)) }}
					</span>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import "../../icons";
import formatter from "../../mixins/formatter";
import Visualization from "./Visualization.vue";
import BatteryIcon from "./BatteryIcon.vue";
import GridIcon from "./GridIcon.vue";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/home";
import "@h2d2/shopicons/es/regular/car3";

export default {
	name: "Energyflow",
	components: { Visualization, BatteryIcon, GridIcon },
	mixins: [formatter],
	props: {
		gridConfigured: Boolean,
		gridPower: { type: Number, default: 0 },
		homePower: { type: Number, default: 0 },
		pvConfigured: Boolean,
		pvPower: { type: Number, default: 0 },
		loadpointsPower: { type: Number, default: 0 },
		activeLoadpointsCount: { type: Number, default: 0 },
		batteryConfigured: Boolean,
		batteryPower: { type: Number, default: 0 },
		batterySoC: { type: Number, default: 0 },
	},
	computed: {
		gridImport: function () {
			return Math.max(0, this.gridPower);
		},
		pvProduction: function () {
			return this.pvConfigured ? Math.abs(this.pvPower) : this.pvExport;
		},
		batteryPowerAdjusted: function () {
			const batteryPowerThreshold = 50;
			return Math.abs(this.batteryPower) < batteryPowerThreshold ? 0 : this.batteryPower;
		},
		batteryDischarge: function () {
			return Math.max(0, this.batteryPowerAdjusted);
		},
		batteryCharge: function () {
			return Math.min(0, this.batteryPowerAdjusted) * -1;
		},
		selfConsumption: function () {
			const ownPower = this.batteryDischarge + this.pvProduction;
			const consumption = this.homePower + this.batteryCharge + this.loadpointsPower;
			return Math.min(ownPower, consumption);
		},
		pvExport: function () {
			return Math.max(0, this.gridPower * -1);
		},
		valuesInKw: function () {
			return this.gridImport + this.selfConsumption + this.pvExport > 1000;
		},
	},
	methods: {
		kw: function (watt) {
			return this.fmtKw(watt, this.valuesInKw);
		},
	},
};
</script>
<style scoped>
.energyflow {
	cursor: pointer;
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
	display: flex;
	justify-content: center;
	width: 50px;
	margin-right: 0.25rem;
	white-space: nowrap;
	flex-shrink: 0;
}
</style>
