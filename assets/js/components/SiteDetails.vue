<template>
	<div class="row">
		<div class="col-6 col-md-3 mt-3" v-if="gridConfigured">
			<div class="mb-2 value" v-if="gridPower > 0">
				Bezug <fa-icon icon="arrow-down" class="text-primary" />
			</div>
			<div class="mb-2 value" v-else>
				Einspeisung <fa-icon icon="arrow-up" class="text-primary"></fa-icon>
			</div>
			<h2 class="value">
				{{ fmt(gridPower) }}
				<small class="text-muted">{{ fmtUnit(gridPower) }}W</small>
			</h2>
		</div>
		<div class="col-6 col-md-3 mt-3" v-if="pvConfigured">
			<div class="mb-2 value">
				Erzeugung
				<fa-icon
					icon="sun"
					:class="{
						'text-primary': pvPower < 0,
						'text-muted': pvPower >= 0,
					}"
				></fa-icon>
			</div>
			<h2 class="value">
				{{ fmt(pvPower) }}
				<small class="text-muted">{{ fmtUnit(pvPower) }}W</small>
			</h2>
		</div>
		<div class="d-md-block col-6 col-md-3 mt-3" v-if="batteryConfigured">
			<div class="mb-2 value">
				Batterie
				<fa-icon class="text-primary" icon="arrow-down" v-if="batteryPower < 0"></fa-icon>
				<fa-icon class="text-primary" icon="arrow-up" v-if="batteryPower > 0"></fa-icon>
			</div>
			<h2 class="value">
				{{ fmt(batteryPower) }}
				<small class="text-muted">{{ fmtUnit(batteryPower) }}W</small>
			</h2>
		</div>
		<div class="col-6 col-md-3 mt-3" v-if="batteryConfigured">
			<div class="mb-2 value">
				SoC
				<fa-icon
					icon="battery-three-quarters"
					:class="{
						'text-primary': batteryPower > 0,
						'text-muted': batteryPower < 0,
					}"
				></fa-icon>
			</div>
			<h2 class="value">{{ batterySoC }} <small class="text-muted">%</small></h2>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "SiteDetails",
	props: {
		gridConfigured: Boolean,
		gridPower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoC: Number,
	},
	mixins: [formatter],
};
</script>
