<template>
	<Site
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
};
</script>
