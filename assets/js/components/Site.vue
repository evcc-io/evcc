<template>
	<div class="flex-grow-1 d-flex flex-column">
		<div class="row mt-4 pt-2">
			<div class="d-none d-md-flex col-12 col-md-3 col-lg-4 align-items-end">
				<p class="h1 text-truncate mb-3">{{ siteTitle || "Home" }}</p>
			</div>
			<div class="col-12 col-md-9 col-lg-8 flex-grow-1">
				<SiteVisualization
					v-if="enableVisualization"
					v-bind="visualization"
				></SiteVisualization>
				<SiteDetails v-else v-bind="details"></SiteDetails>
			</div>
		</div>
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
