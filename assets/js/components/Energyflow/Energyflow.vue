<template>
	<div>
		<div
			class="row align-items-start align-items-md-center mt-4 energyflow"
			@click="toggleDetails"
		>
			<Visualization
				class="col-12 offset-md-1 col-md-6 offset-lg-1 col-lg-8 offset-xl-1 col-xl-6 offset-xxl-1 col-xl-8 order-md-2"
				:showDetails="showDetails"
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
			<div
				class="col-12 col-sm-6 col-md-5 col-lg-3 col-xl-3 order-md-1 mt-2 mt-md-0"
				:class="`${showDetails ? 'd-block' : `d-none d-md-block`}`"
			>
				<div class="d-flex justify-content-between" data-test-pv-production>
					<span class="details-icon text-muted"><fa-icon icon="sun"></fa-icon></span>
					<span class="text-nowrap flex-grow-1">{{
						$t("main.energyflow.pvProduction")
					}}</span>
					<span class="text-end text-nowrap ps-1">{{ kw(pvProduction) }}</span>
				</div>
				<div class="d-flex justify-content-between" data-test-home-power>
					<span class="details-icon text-muted"><fa-icon icon="home"></fa-icon></span>
					<span class="text-nowrap flex-grow-1">{{
						$t("main.energyflow.homePower")
					}}</span>
					<span class="text-end text-nowrap ps-1">{{ kw(homePower) }}</span>
				</div>
				<div class="d-flex justify-content-between" data-test-loadpoints>
					<span class="details-icon text-muted"><fa-icon icon="car"></fa-icon></span>
					<span class="text-nowrap flex-grow-1">{{
						$tc("main.energyflow.loadpoints", activeLoadpointsCount, {
							count: activeLoadpointsCount,
						})
					}}</span>
					<span class="text-end text-nowrap ps-1">{{ kw(loadpointsPower) }}</span>
				</div>
				<div
					v-if="batteryConfigured"
					class="d-flex justify-content-between"
					data-test-battery
				>
					<span class="details-icon text-muted">
						<BatteryIcon
							:soc="batterySoC"
							:charge="batteryCharge > 0"
							:discharge="batteryDischarge > 0"
						/>
					</span>
					<span class="text-nowrap flex-grow-1 text-truncate">
						<span v-if="batteryCharge">{{ $t("main.energyflow.batteryCharge") }}</span>
						<span v-else-if="batteryDischarge">{{
							$t("main.energyflow.batteryDischarge")
						}}</span>
						<span v-else>{{ $t("main.energyflow.battery") }}</span>
					</span>
					<span class="text-end text-nowrap ps-1">
						{{ batterySoC }}% /
						{{ kw(Math.abs(batteryPower)) }}
					</span>
				</div>
			</div>
			<div
				v-if="showDetails"
				class="col-12 col-sm-6 offset-md-6 col-md-6 offset-lg-4 col-lg-8 d-block d-md-flex order-md-3 justify-content-between mt-2"
			>
				<div class="text-nowrap d-flex d-md-block" data-test-grid-import>
					<span class="color-grid details-icon"><fa-icon icon="square"></fa-icon></span>
					<span class="text-nowrap flex-grow-1">{{
						$t("main.energyflow.gridImport")
					}}</span>
					<span class="text-end text-nowrap d-md-none">
						{{ kw(gridImport) }}
					</span>
				</div>
				<div class="text-nowrap d-flex d-md-block" data-test-self-consumption>
					<span class="color-self details-icon"><fa-icon icon="square"></fa-icon></span>
					<span class="text-nowrap flex-grow-1">{{
						$t("main.energyflow.selfConsumption")
					}}</span>
					<span class="text-end text-nowrap d-md-none">
						{{ kw(selfConsumption) }}
					</span>
				</div>
				<div class="text-nowrap d-flex d-md-block" data-test-pv-export>
					<span class="color-export details-icon"><fa-icon icon="square"></fa-icon></span>
					<span class="text-nowrap flex-grow-1">{{
						$t("main.energyflow.pvExport")
					}}</span>
					<span class="text-end text-nowrap d-md-none">
						{{ kw(pvExport) }}
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

export default {
	name: "Energyflow",
	components: { Visualization, BatteryIcon },
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
	data: function () {
		return { showDetails: false };
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
		toggleDetails() {
			this.showDetails = !this.showDetails;
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
	text-align: center;
	width: 30px;
	margin-right: 0.25rem;
	white-space: nowrap;
	flex-shrink: 0;
}
</style>
