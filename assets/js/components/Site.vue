<template>
	<div class="flex-grow-1 d-flex flex-column">
		<SiteVisualization v-if="enableVisualization" v-bind="visualization" />
		<hr class="w-100 my-4" />
		<div class="flex-grow-1 d-flex justify-content-around flex-column">
			<template v-for="(loadpoint, id) in loadpoints">
				<hr class="w-100 my-4" v-if="id > 0" :key="id + '_hr'" />
				<Loadpoint
					:key="id"
					v-bind="loadpoint"
					:single="loadpoints.length === 1"
					:id="id"
				/>
			</template>
		</div>
	</div>
</template>

<script>
import SiteDetails from "./SiteDetails";
import SiteVisualization from "./SiteVisualization";
import Loadpoint from "./Loadpoint";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";

export default {
	name: "Site",
	props: {
		siteTitle: String,
		loadpoints: Array,

		// details
		gridConfigured: Boolean,
		gridPower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoC: Number,
		gridCurrents: Array,
		prioritySoC: Number,
	},
	data: function () {
		return { enableVisualization: true };
	},
	components: { SiteDetails, Loadpoint, SiteVisualization },
	mixins: [formatter, collector],
	computed: {
		details: function () {
			return this.collectProps(SiteDetails);
		},
		visualization: function () {
			return this.collectProps(SiteVisualization);
		},
	},
};
</script>
