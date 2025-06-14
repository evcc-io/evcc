<template>
	<Site
		v-if="state.startup"
		:notifications="notifications"
		v-bind="state"
		:selected-loadpoint-index="selectedLoadpointIndex"
	/>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Site from "../components/Site/Site.vue";
import store from "../store";
import type { Notification } from "@/types/evcc";

export default defineComponent({
	name: "Main",
	components: { Site },
	props: {
		notifications: Array as PropType<Notification[]>,
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
});
</script>
