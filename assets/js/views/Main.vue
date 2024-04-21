<template>
	<Site :notifications="notifications" v-bind="state"></Site>
</template>

<script>
import Site from "../components/Site.vue";
import store from "../store";

export default {
	name: "Main",
	components: { Site },
	props: {
		notifications: Array,
	},
	data: function () {
		return store;
	},
};
</script>
