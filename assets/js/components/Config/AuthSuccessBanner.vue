<template>
	<div class="alert alert-success my-4 pb-0" role="alert" data-testid="auth-success-banner">
		<p>
			{{ $t("authProviders.success", { title: providerName }) }}
			{{ $t("authProviders.successCloseTab") }}
		</p>
	</div>
</template>

<script lang="ts">
import type { AuthProviders } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "AuthSuccessBanner",
	props: {
		providerId: { type: String, required: true },
		authProviders: { type: Object as PropType<AuthProviders>, default: () => ({}) },
	},
	computed: {
		providerName() {
			for (const [name, provider] of Object.entries(this.authProviders)) {
				if (provider.id === this.providerId) {
					return name;
				}
			}
			return "Unknown";
		},
	},
});
</script>
