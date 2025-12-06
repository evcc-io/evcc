<template>
	<button
		v-if="hasUnauthenticated"
		type="button"
		class="btn btn-sm btn-link text-decoration-none border-0 text-nowrap auth-indicator"
		data-testid="auth-indicator"
		@click="openAuthModal"
	>
		<KeyIcon class="text-warning" />
	</button>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import KeyIcon from "../MaterialIcon/Key.vue";
import type { AuthProviders } from "@/types/evcc";

export default defineComponent({
	name: "AuthIndicator",
	components: {
		KeyIcon,
	},
	props: {
		authProviders: {
			type: Object as PropType<AuthProviders>,
			default: () => ({}),
		},
	},
	emits: ["auth-required"],
	computed: {
		firstUnauthenticatedProvider() {
			const entry = Object.entries(this.authProviders).find(
				// eslint-disable-next-line @typescript-eslint/no-unused-vars
				([_title, provider]) => !provider.authenticated
			);
			if (!entry) return null;
			const [title, { authenticated, id }] = entry;
			return { title, authenticated, id };
		},
		hasUnauthenticated(): boolean {
			return this.firstUnauthenticatedProvider !== null;
		},
	},
	methods: {
		openAuthModal() {
			if (this.firstUnauthenticatedProvider) {
				this.$emit("auth-required", this.firstUnauthenticatedProvider);
			}
		},
	},
});
</script>

<style scoped>
.auth-indicator {
	padding: 0.25rem 0.5rem;
}
</style>
