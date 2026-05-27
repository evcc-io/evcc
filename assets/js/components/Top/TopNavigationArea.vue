<template>
	<div class="d-flex gap-2">
		<Notifications
			:notifications="notifications"
			:loadpoints="loadpoints"
			class="d-flex align-items-center"
		/>
		<Savings v-bind="savings" />
		<AuthProviderModal :provider="authProvider" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Modal from "bootstrap/js/dist/modal";
import Notifications from "./Notifications.vue";
import AuthProviderModal from "./AuthProviderModal.vue";
import Savings from "../Savings/Savings.vue";
import type { Provider } from "./types";
import type { Notification } from "@/types/evcc";
import store from "@/store";

export default defineComponent({
	name: "TopNavigationArea",
	components: {
		Notifications,
		AuthProviderModal,
		Savings,
	},
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
		savings() {
			return {
				sponsor: store.state.sponsor,
				statistics: store.state.statistics,
				co2Configured: store.state.tariffCo2 !== undefined,
				currency: store.state.currency,
				telemetry: store.state.telemetry,
				forecast: store.state.forecast,
				tariffGrid: store.state.tariffGrid,
			};
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
