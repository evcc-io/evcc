<template>
	<div class="mt-2">
		<div class="row">
			<div class="d-none d-md-flex col-12 col-md-3 col-lg-6 align-items-end">
				<p class="h2">{{ siteTitle || "Home" }}</p>
			</div>
			<div class="col-12 col-md-9 col-lg-6">
				<SiteDetails v-bind="details"></SiteDetails>
			</div>
		</div>

		<Loadpoint
			v-for="(loadpoint, id) in loadpoints"
			v-bind="loadpoint"
			:single="loadpoints.length === 1"
			:id="id"
			:key="id"
			:pvConfigured="pvConfigured"
		/>
	</div>
</template>

<script>
import SiteDetails from "./SiteDetails";
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
	},
	components: { SiteDetails, Loadpoint },
	mixins: [formatter, collector],
	computed: {
		details: function () {
			return this.collectProps(SiteDetails);
		},
	},
};
</script>
