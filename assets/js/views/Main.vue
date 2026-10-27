<template>
	<Site
		v-if="!wizardPending"
		:notifications="notifications"
		v-bind="state"
		:forecast="uiForecast"
		:selected-loadpoint-index="selectedLoadpointIndex"
	/>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Site from "../components/Site/Site.vue";
import store from "../store";
import type { Notification } from "@/types/evcc";
import { wizardPending } from "@/onboarding";

export default defineComponent({
	name: "Main",
	components: { Site },
	props: {
		notifications: Array as PropType<Notification[]>,
		selectedLoadpointIndex: Number,
	},
	computed: {
		state() {
			return store.state;
		},
		uiForecast() {
			return store.uiForecast.value;
		},
		wizardPending() {
			return wizardPending();
		},
	},
	head() {
		const title = store.state.siteTitle;
		if (title) {
			return {
				title,
				meta: [
					{ name: "apple-mobile-web-app-title", content: title },
					{ name: "application-name", content: title },
				],
			};
		}
		// no custom title
		return { title: "evcc", titleTemplate: null };
	},
	watch: {
		wizardPending: {
			immediate: true,
			handler(pending: boolean) {
				if (pending) this.$router.replace("/config");
			},
		},
		'state.siteTitle': {
			handler(newTitle) {
				// Only create manifest when we have the final configured title
				// Skip App.vue's temporary "..." placeholder
				if (newTitle && newTitle !== "...") {
					this.updateManifest();
				}
			},
		},
	},
	mounted() {
		// Only create manifest when final title is loaded
		// Skip if still showing "..." placeholder from App.vue
		const title = store.state.siteTitle;
		if (title && title !== "...") {
			this.updateManifest();
		}
	},
	methods: {
		updateManifest() {
			// Use configured title or fallback to "evcc"
			// siteTitle from store does NOT contain the " | evcc" suffix
			// (that suffix is only added in App.vue for document.title)
			const title = store.state.siteTitle || "evcc";

			// Remove old manifest if present
			const oldLink = document.querySelector('link[rel="manifest"]');
			if (oldLink) {
				oldLink.remove();
			}

			// Create new manifest with custom title
			const manifest = {
				name: title,
				short_name: title,
				start_url: "/",
				display: "standalone",
				theme_color: "#1d1d1b",
				background_color: "#1d1d1b",
				icons: [
					{
						src: "/meta/android-chrome-192x192.png",
						sizes: "192x192",
						type: "image/png"
					},
					{
						src: "/meta/android-chrome-512x512.png",
						sizes: "512x512",
						type: "image/png"
					}
				]
			};

			const blob = new Blob([JSON.stringify(manifest)], { type: 'application/json' });
			const manifestURL = URL.createObjectURL(blob);

			const link = document.createElement('link');
			link.rel = 'manifest';
			link.href = manifestURL;
			document.head.appendChild(link);
		},
	},
});
</script>
