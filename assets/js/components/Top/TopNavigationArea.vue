<template>
	<div class="d-flex">
		<Notifications :notifications="notifications" :loadpoints="loadpoints" class="me-2" />
		<TopNavigation v-bind="topNavigation" @auth-required="openAuthModal" />
		<AuthProviderModal :provider="authProvider" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Modal from "bootstrap/js/dist/modal";
import Navigation from "./Navigation.vue";
import Notifications from "./Notifications.vue";
import AuthProviderModal from "./AuthProviderModal.vue";
import type { Provider } from "./types";
import type { Notification } from "@/types/evcc";
import collector from "@/mixins/collector";
import store from "@/store";

export default defineComponent({
	name: "TopNavigationArea",
	components: {
		TopNavigation: Navigation,
		Notifications,
		AuthProviderModal,
	},
	mixins: [collector],
	props: {
		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
	},
	data() {
		return {
			authProviderId: null as string | null,
			isUnmounted: false,
		};
	},
	computed: {
		topNavigation(): any {
			return this.collectProps(Navigation, store.state);
		},
		loadpoints() {
			return store.uiLoadpoints.value || [];
		},
		authProviders() {
			return store.state?.authProviders || {};
		},
		authProvider(): Provider | null {
			if (!this.authProviderId) return null;
			const entry = Object.entries(this.authProviders).find(
				([, value]) => value.id === this.authProviderId
			);
			if (!entry) return null;
			const [title, { id, authenticated }] = entry;
			return {
				id,
				title,
				authenticated,
			};
		},
	},
	unmounted() {
		this.isUnmounted = true;
	},
	methods: {
		openAuthModal(providerId: string) {
			this.authProviderId = providerId;
			this.$nextTick(() => {
				if (this.isUnmounted) return;
				const modalElement = document.getElementById("authProviderModal");
				if (!modalElement) return;
				const modal = Modal.getOrCreateInstance(modalElement);
				modal?.show();
			});
		},
		// Public method for imperative calls (e.g., from Config page)
		requestAuthProvider(providerId: string) {
			this.openAuthModal(providerId);
		},
	},
});
</script>
