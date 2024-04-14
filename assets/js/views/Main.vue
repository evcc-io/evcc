<template>
<<<<<<< HEAD
	<div>
		<StartupError v-if="startupErrors" v-bind="state" :offline="offline" />
		<Site v-else :notifications="notifications" v-bind="state"></Site>
	</div>
=======
	<Site :notifications="notifications" v-bind="state"></Site>
>>>>>>> 334ffc15b (Log Viewer: Allow retrieving logs from ring buffer (#13330))
</template>

<script>
import Site from "../components/Site.vue";
<<<<<<< HEAD
import StartupError from "../components/StartupError.vue";
=======
>>>>>>> 334ffc15b (Log Viewer: Allow retrieving logs from ring buffer (#13330))
import store from "../store";

export default {
	name: "Main",
<<<<<<< HEAD
	components: { Site, StartupError },
=======
	components: { Site },
>>>>>>> 334ffc15b (Log Viewer: Allow retrieving logs from ring buffer (#13330))
	props: {
		notifications: Array,
	},
	data: function () {
		return store;
	},
};
</script>
