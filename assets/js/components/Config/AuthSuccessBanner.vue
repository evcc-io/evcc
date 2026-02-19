<template>
	<div
		class="alert my-4 pb-0"
		:class="isError ? 'alert-danger' : 'alert-success'"
		role="alert"
		:data-testid="testId"
	>
		<p v-if="isError">
			<strong>{{ $t("authProviders.authorizationFailed") }}</strong
			><br />
			{{ errorMessage }}
		</p>
		<p v-else>
			<strong>{{ $t("authProviders.authorizationSuccessful") }}</strong
			><br />
			{{ $t("authProviders.success", { title: providerName }) }}
		</p>
	</div>
</template>

<script lang="ts">
import type { AuthProviders } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "AuthSuccessBanner",
	props: {
		providerId: { type: String, default: "" },
		error: { type: String, default: "" },
		authProviders: { type: Object as PropType<AuthProviders>, default: () => ({}) },
	},
	computed: {
		isError() {
			return !!this.error;
		},
		errorMessage() {
			return this.error || "";
		},
		providerName() {
			for (const [name, provider] of Object.entries(this.authProviders)) {
				if (provider.id === this.providerId) {
					return name;
				}
			}
			return "Unknown";
		},
		testId() {
			return this.isError ? "auth-error-banner" : "auth-success-banner";
		},
	},
});
</script>
