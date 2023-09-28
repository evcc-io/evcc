<template>
	<div>
		<OfflineIndicator v-if="offline" />
		<StartupError v-if="startupErrors" v-bind="state" :offline="offline" />
		<Site v-else :notifications="notifications" v-bind="state"></Site>
	</div>
</template>

<script>
import Site from "../components/Site.vue";
import StartupError from "../components/StartupError.vue";
import OfflineIndicator from "../components/OfflineIndicator.vue";
import store from "../store";

export default {
	name: "Main",
	components: { Site, StartupError, OfflineIndicator },
	props: {
		notifications: Array,
		offline: Boolean,
	},
	data: function () {
		return store;
	},
	computed: {
		startupErrors: function () {
			return this.state.fatal?.length > 0;
		},
	},
};
</script>
