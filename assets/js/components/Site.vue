<template>
	<div>
		<div class="row">
			<div class="d-none d-md-flex col-12 col-md-4 mt-md-4 align-items-end">
				<p class="h1">{{ title || "Home" }}</p>
			</div>
			<div class="col-12 col-md-8 mt-md-4" v-if="multi">
				<SiteDetails v-bind="details"></SiteDetails>
			</div>
		</div>

		<div class="row d-none d-md-flex border-bottom"></div>

		<div class="row" v-if="!multi">
			<div class="d-none d-md-block col-md-4"></div>
			<div class="col-12 col-md-8">
				<SiteDetails v-bind="details"></SiteDetails>
			</div>
		</div>

		<Loadpoint
			v-for="(loadpoint, id) in loadpoints"
			v-bind="loadpoint"
			:id="id"
			:key="id"
			:multi="multi"
			:pvConfigured="pvConfigured"
		>
		</Loadpoint>
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
		title: String,
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
		multi: function () {
			// TODO fix compact
			return this.loadpoints.length > 1 /* || app.compact*/;
		},
	},
};
</script>
