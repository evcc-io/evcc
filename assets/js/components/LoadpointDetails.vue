<template>
	<div>
		<div class="row">
			<div class="col-6 col-sm-3 col-lg-2 mt-3 offset-lg-4">
				<div class="mb-2 value">
					{{ $t("main.loadpointDetails.power") }}
					<fa-icon
						class="text-primary ms-1"
						icon="temperature-low"
						v-if="climater == 'heating'"
					></fa-icon>
					<fa-icon
						class="text-primary ms-1"
						icon="temperature-high"
						v-if="climater == 'cooling'"
					></fa-icon>
					<fa-icon
						class="text-primary ms-1"
						icon="thermometer-half"
						v-if="climater == 'on'"
					></fa-icon>
				</div>
				<h3 class="value">
					{{ fmt(chargePower) }}
					<small class="text-muted">{{ fmtUnit(chargePower) }}W</small>
				</h3>
			</div>

			<div class="col-6 col-sm-3 col-lg-2 mt-3">
				<div class="mb-2 value">{{ $t("main.loadpointDetails.charged") }}</div>
				<h3 class="value">
					{{ fmt(chargedEnergy) }}
					<small class="text-muted">{{ fmtUnit(chargedEnergy) }}Wh</small>
				</h3>
			</div>

			<div class="col-6 col-sm-3 col-lg-2 mt-3" v-if="vehicleRange && vehicleRange >= 0">
				<div class="mb-2 value">{{ $t("main.loadpointDetails.vehicleRange") }}</div>
				<h3 class="value">
					{{ Math.round(vehicleRange) }}
					<small class="text-muted">km</small>
				</h3>
			</div>

			<div class="col-6 col-sm-3 col-lg-2 mt-3" v-else>
				<div class="mb-2 value">{{ $t("main.loadpointDetails.duration") }}</div>
				<h3 class="value">
					{{ fmtShortDuration(chargeDuration) }}
					<small class="text-muted">{{ fmtShortDurationUnit(chargeDuration) }}</small>
				</h3>
			</div>

			<div class="col-6 col-sm-3 col-lg-2 mt-3" v-if="vehiclePresent">
				<div class="mb-2 value">{{ $t("main.loadpointDetails.remaining") }}</div>
				<h3 class="value">
					{{ fmtShortDuration(chargeRemainingDuration) }}
					<small class="text-muted">{{
						fmtShortDurationUnit(chargeRemainingDuration)
					}}</small>
				</h3>
			</div>
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
		chargeRemainingDuration: Number,
		chargePower: Number,
		climater: String,
		vehiclePresent: Boolean,
		vehicleRange: Number,
	},
	mixins: [formatter],
};
</script>
