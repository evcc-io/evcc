<template>
	<div class="flex-grow-1 d-flex flex-column">
		<div class="container">
			<h3 class="d-none d-md-block my-4">
				{{ siteTitle || "Home" }}
			</h3>
			<Energyflow v-bind="energyflow" />
		</div>
		<div class="flex-grow-1 d-flex justify-content-around flex-column content-area">
			<div class="container">
				<template v-for="(loadpoint, id) in loadpoints">
					<hr v-if="id > 0" :key="id + '_hr'" class="w-100 my-4" />
					<Loadpoint
						v-bind="loadpoint"
						:id="id"
						:key="id"
						:single="loadpoints.length === 1"
					/>
				</template>
			</div>
		</div>
	</div>
</template>

<script>
import Energyflow from "./Energyflow";
import Loadpoint from "./Loadpoint";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";

export default {
	name: "Site",
	components: { Loadpoint, Energyflow },
	mixins: [formatter, collector],
	props: {
		loadpoints: Array,

		// details
		gridConfigured: Boolean,
		gridPower: Number,
		homePower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoC: Number,
		gridCurrents: Array,
		prioritySoC: Number,
		siteTitle: String,
	},
	computed: {
		energyflow: function () {
			return this.collectProps(Energyflow);
		},
		activeLoadpointsCount: function () {
			return this.loadpoints.filter((lp) => lp.chargePower > 0).length;
		},
		loadpointsPower: function () {
			return this.loadpoints.reduce((sum, lp) => {
				sum += lp.chargePower || 0;
				return sum;
			}, 0);
		},
	},
};
</script>
<style scoped>
.content-area {
	background-color: var(--bs-gray-dark);
	border-radius: 20px 20px 0 0;
	color: var(--bs-white);
}
</style>
