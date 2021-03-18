<template>
	<div class="row">
		<div class="col-6 col-md-3 mt-3">
			<div class="mb-2 value">
				Leistung
				<fa-icon
					class="text-primary ml-1"
					icon="temperature-low"
					v-if="climater == 'heating'"
				></fa-icon>
				<fa-icon
					class="text-primary ml-1"
					icon="temperature-high"
					v-if="climater == 'cooling'"
				></fa-icon>
				<fa-icon
					class="text-primary ml-1"
					icon="thermometer-half"
					v-if="climater == 'on'"
				></fa-icon>
			</div>
			<h2 class="value">
				{{ fmt(chargePower) }}
				<small class="text-muted">{{ fmtUnit(chargePower) }}W</small>
			</h2>
		</div>
		<div class="col-6 col-md-3 mt-3">
			<div class="mb-2 value">Geladen</div>
			<h2 class="value">
				{{ fmt(chargedEnergy) }}
				<small class="text-muted">{{ fmtUnit(chargedEnergy) }}Wh</small>
			</h2>
		</div>

		<div class="col-6 col-md-3 mt-3" v-if="range >= 0">
			<div class="mb-2 value">Reichweite</div>
			<h2 class="value">
				{{ Math.round(range) }}
				<small class="text-muted">km</small>
			</h2>
		</div>

		<div class="col-6 col-md-3 mt-3" v-else>
			<div class="mb-2 value">Dauer</div>
			<h2 class="value">
				{{ fmtShortDuration(chargeDuration) }}
				<small class="text-muted">{{ fmtShortDurationUnit(chargeDuration) }}</small>
			</h2>
		</div>

		<div class="col-6 col-md-3 mt-3" v-if="hasVehicle">
			<div class="mb-2 value">Restzeit</div>
			<h2 class="value">
				{{ fmtShortDuration(chargeEstimate) }}
				<small class="text-muted">{{ fmtShortDurationUnit(chargeEstimate) }}</small>
			</h2>
		</div>
	</div>
</template>

<script>
import "../icons";
import formatter from "../mixins/formatter";

export default {
	name: "LoadpointDetails",
	props: {
		chargedEnergy: Number,
		chargeDuration: Number,
		chargeEstimate: Number,
		chargePower: Number,
		climater: String,
		hasVehicle: Boolean,
		range: Number,
	},
	mixins: [formatter],
};
</script>
