<template>
	<div class="row">
		<div class="col-6 col-md-3 mt-3" v-if="state.gridConfigured">
			<div class="mb-2 value" v-if="state.gridPower > 0">
				Bezug <font-awesome-icon icon="arrow-down" class="text-primary" />
			</div>
			<div class="mb-2 value" v-else>
				Einspeisung <font-awesome-icon icon="arrow-up" class="text-primary" />
			</div>
			<h2 class="value">
				{{ fmt(state.gridPower) }}
				<small class="text-muted">{{ fmtUnit(state.gridPower) }}W</small>
			</h2>
		</div>
		<div class="col-6 col-md-3 mt-3" v-if="state.pvConfigured">
			<div class="mb-2 value">
				Erzeugung
				<font-awesome-icon
					icon="sun"
					v-bind:class="{
						'text-primary': state.pvPower < 0,
						'text-muted': state.pvPower >= 0,
					}"
				/>
			</div>
			<h2 class="value">
				{{ fmt(state.pvPower) }}
				<small class="text-muted">{{ fmtUnit(state.pvPower) }}W</small>
			</h2>
		</div>
		<div
			class="d-md-block col-6 col-md-3 mt-3"
			v-bind:class="{ 'd-none': !state.batterySoC }"
			v-if="state.batteryConfigured"
		>
			<div class="mb-2 value">
				Batterie
				<font-awesome-icon
					class="text-primary"
					icon="arrow-down"
					v-if="state.batteryPower < 0"
				/>
				<font-awesome-icon
					class="text-primary"
					icon="arrow-up"
					v-if="state.batteryPower > 0"
				/>
			</div>
			<h2 class="value">
				{{ fmt(state.batteryPower) }}
				<small class="text-muted">{{ fmtUnit(state.batteryPower) }}W</small>
			</h2>
		</div>
		<div class="col-6 col-md-3 mt-3" v-if="state.batterySoC">
			<div class="mb-2 value">
				SoC
				<font-awesome-icon
					class="text-primary"
					icon="battery-three-quarters"
					v-bind:class="{
						'text-primary': state.batteryPower > 0,
						'text-muted': state.batteryPower < 0,
					}"
				/>
			</div>
			<h2 class="value">{{ state.batterySoC }} <small class="text-muted">%</small></h2>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "SiteDetails",
	props: ["state"],
	mixins: [formatter],
};
</script>
