<template>
	<div>
		<div class="row">
			<div class="d-none d-md-flex col-12 col-md-4 col-lg-3 mt-md-4 align-items-end">
				<p class="h1">{{ siteTitle || "Home" }}</p>
			</div>
			<div class="col-12 col-md-8 col-lg-9 mt-3" v-if="multi">
				<SiteDetails v-bind="details"></SiteDetails>
			</div>
		</div>

		<Loadpoint
			v-for="(loadpoint, id) in loadpoints"
			v-bind="loadpoint"
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
