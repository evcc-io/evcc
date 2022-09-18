<template>
	<div>
		<Site
			v-if="configured"
			:notifications="notifications"
			:offline="offline"
			v-bind="state"
		></Site>
		<div v-else class="container">
			<div class="row py-5">
				<div v-for="(error, index) in errors" :key="`err-${index}`" class="col12">
					<span v-if="index == 0">Fehler:</span>
					<span v-else>Ursache:</span>
					<code>{{ error }}</code>
				</div>
			</div>
			<div class="row py-5">
				<div class="col12">File:{{ file }}</div>
				<div class="col12">Line:{{ line }}</div>
			</div>
			<div class="row py-5">
				<div class="col12">Config:</div>
				<div class="col12">
					<code v-if="config">
						<pre>{{ config }}</pre>
					</code>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import Site from "../components/Site.vue";
import store from "../store";

export default {
	name: "Main",
	components: { Site },
	props: {
		notifications: Array,
		offline: Boolean,
	},
	data: function () {
		return store;
	},
	computed: {
		configured: function () {
			return this.errors.length === 0;
		},
		errors: function () {
			return this.state.fatal || [];
		},
		config: function () {
			return this.state.config;
		},
		file: function () {
			return this.state.file;
		},
		line: function () {
			return this.state.line;
		},
	},
};
</script>
