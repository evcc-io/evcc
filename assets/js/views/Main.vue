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
					<span v-else>verursacht durch:</span>
					<code>{{ error }}</code>
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
			return this.state.fatal === null || this.state.fatal.length === 0;
		},
		errors: function () {
			return this.state.fatal === null ? [] : this.state.fatal;
		},
	},
};
</script>
