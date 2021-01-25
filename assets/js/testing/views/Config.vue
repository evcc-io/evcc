<template>
	<div class="container">
		<ul class="nav nav-tabs" id="myTab" role="tablist">
			<li class="nav-item" role="presentation" v-for="(val, key) in tabs" :key="key">
				<a
					class="nav-link"
					data-toggle="tab"
					role="tab"
					v-bind:href="'#' + key"
					:class="{ active: key == 'meter' }"
					:id="key + '-tab'"
					>{{ val }}</a
				>
			</li>
		</ul>

		<div class="tab-content" id="myTabContent">
			<div
				class="tab-pane fade show"
				role="tabpanel"
				v-for="(val, key) in tabs"
				:class="{ active: key == 'meter' }"
				:id="key"
				:key="key"
			>
				<ConfigClass :klass="key" :plugins="plugins"></ConfigClass>
			</div>
		</div>

		<!-- <div>
			<Ssh></Ssh>
		</div> -->
	</div>
</template>

<script>
import axios from "axios";
import ConfigClass from "../components/ConfigClass";
// import Configurable from "../components/Configurable";
// import Ssh from "../components/Ssh";

export default {
	name: "Config",
	components: { ConfigClass },
	data: function () {
		return {
			plugins: [],
		};
	},
	computed: {
		tabs: function () {
			return {
				meter: "Meter",
				charger: "Charger",
				vehicle: "Vehicle",
				plugin: "Plugin",
			};
		},
	},
	mounted: async function () {
		try {
			this.plugins = (await axios.get("/config/types/plugin")).data;
		} catch (e) {
			window.toasts.error(e);
		}
	},
};
</script>
