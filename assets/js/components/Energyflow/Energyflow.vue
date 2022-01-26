<template>
	<div class="energyflow">
		<div class="row">
			<Visualization
				class="col-12 mb-3"
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
		</div>
		<div class="details">
			<div class="details-inner row" data-collapsible-details>
				<div class="col-12 d-flex justify-content-between pt-2 mb-4">
					<div class="d-flex flex-nowrap">
						<span class="color-self me-2"><fa-icon icon="square"></fa-icon></span>
						<span>{{ $t("main.energyflow.selfConsumption") }}</span>
					</div>
					<div v-if="gridImport > 0" class="d-flex flex-nowrap">
						<span>{{ $t("main.energyflow.gridImport") }}</span>
						<span class="color-grid ms-2"><fa-icon icon="square"></fa-icon></span>
					</div>
					<div v-else class="d-flex flex-nowrap">
						<span>{{ $t("main.energyflow.pvExport") }}</span>
						<span class="color-export ms-2"><fa-icon icon="square"></fa-icon></span>
					</div>
				</div>
				<div
					class="col-12 col-md-6 pe-md-5 pb-4 d-flex flex-column justify-content-between"
				>
					<div class="d-flex justify-content-between align-items-end mb-4">
						<h3 class="m-0">In</h3>
						<span class="fw-bold opacity-25">{{ kw(inPower) }}</span>
					</div>
					<div>
						<EnergyflowEntry
							:name="$t('main.energyflow.pvProduction')"
							icon="sun"
							:power="pvProduction"
							:valuesInKw="valuesInKw"
							type="source"
						/>
						<EnergyflowEntry
							v-if="batteryConfigured"
							:name="$t('main.energyflow.batteryDischarge')"
							icon="battery"
							:soc="batterySoC"
							:power="batteryDischarge"
							:valuesInKw="valuesInKw"
							type="source"
						/>
						<EnergyflowEntry
							:name="$t('main.energyflow.gridImport')"
							icon="powersupply"
							:power="gridImport"
							:valuesInKw="valuesInKw"
							type="source"
						/>
					</div>
				</div>
				<div
					class="col-12 col-md-6 ps-md-5 pb-4 d-flex flex-column justify-content-between"
				>
					<div class="d-flex justify-content-between align-items-end mb-4">
						<h3 class="m-0">Out</h3>
						<span class="fw-bold opacity-25">{{ kw(outPower) }}</span>
					</div>
					<div>
						<EnergyflowEntry
							:name="$t('main.energyflow.homePower')"
							icon="home"
							:power="homePower"
							:valuesInKw="valuesInKw"
							type="consumer"
						/>
						<EnergyflowEntry
							:name="
								$tc('main.energyflow.loadpoints', activeLoadpointsCount, {
									count: activeLoadpointsCount,
								})
							"
							icon="car3"
							:power="loadpointsPower"
							:valuesInKw="valuesInKw"
							type="consumer"
						/>
						<EnergyflowEntry
							v-if="batteryConfigured"
							:name="$t('main.energyflow.batteryCharge')"
							icon="battery"
							:soc="batterySoC"
							:power="batteryCharge"
							:valuesInKw="valuesInKw"
							type="consumer"
						/>
						<EnergyflowEntry
							:name="$t('main.energyflow.pvExport')"
							icon="powersupply"
							:power="pvExport"
							:valuesInKw="valuesInKw"
							type="consumer"
						/>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import Visualization from "./Visualization.vue";
import EnergyflowEntry from "./EnergyflowEntry.vue";
import formatter from "../../mixins/formatter";

export default {
	name: "Energyflow",
	components: { Visualization, EnergyflowEntry },
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
			return Math.abs(Math.max(0, this.batteryPowerAdjusted));
		},
		batteryCharge: function () {
			return Math.abs(Math.min(0, this.batteryPowerAdjusted) * -1);
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
		inPower: function () {
			return this.gridImport + this.pvProduction + this.batteryDischarge;
		},
		outPower: function () {
			return this.homePower + this.loadpointsPower + this.pvExport + this.batteryCharge;
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
	background: var(--bs-white);
}
.details {
	height: 0;
	overflow: visible;
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
