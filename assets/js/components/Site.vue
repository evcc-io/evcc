<template>
	<div>
		<div class="row">
			<div class="d-none d-md-flex col-12 col-md-4 mt-md-4 align-items-end">
				<p class="h1">{{ state.title || "Home" }}</p>
			</div>
			<div class="col-12 col-md-8 mt-md-4" v-if="multi">
				<SiteDetails v-bind:state="state"></SiteDetails>
			</div>
		</div>

		<div class="row d-none d-md-flex border-bottom"></div>

		<div class="row" v-if="!multi">
			<div class="d-none d-md-block col-md-4"></div>
			<div class="col-12 col-md-8">
				<SiteDetails v-bind:state="state"></SiteDetails>
			</div>
		</div>

		<Loadpoint
			v-for="(loadpoint, id) in state.loadpoints"
			v-bind:id="id"
			:key="id"
			:state="loadpoint"
			:pv="state.gridConfigured"
			:multi="multi"
		>
		</Loadpoint>
	</div>
</template>

<script>
import SiteDetails from "./SiteDetails";
import Loadpoint from "./Loadpoint";
import formatter from "../mixins/formatter";

export default {
	name: "Site",
	props: ["state"],
	components: { SiteDetails, Loadpoint },
	mixins: [formatter],
	computed: {
		multi: function () {
			// TODO fix compact
			return this.state.loadpoints.length > 1 /* || app.compact*/;
		},
	},
};
</script>
