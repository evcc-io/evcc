<template>
	<Site
		v-if="state.startup"
		:notifications="notifications"
		v-bind="state"
		:selected-loadpoint-index="selectedLoadpointIndex"
	/>
</template>

<script>
import Site from "../components/Site/Site.vue";
import store from "../store";

export default {
	name: "Main",
	components: { Site },
	props: {
		notifications: Array,
		selectedLoadpointIndex: Number,
	},
	data() {
		return store;
	},
	head() {
		const title = store.state.siteTitle;
		if (title) {
			return { title };
		}
		// no custom title
		return { title: "evcc", titleTemplate: null };
	},
};
</script>
