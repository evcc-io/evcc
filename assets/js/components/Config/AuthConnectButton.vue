<template>
	<!-- Connect to provider button (when URL is available) -->
	<div v-if="providerUrl" class="d-flex flex-column align-items-end gap-2">
		<a :href="providerUrl" target="_blank" class="btn btn-primary">
			{{
				$t("authProviders.buttonConnect", {
					provider: providerDomain,
				})
			}}
		</a>
		<small v-if="showHint" class="d-block">{{ $t("config.general.authPerformHint") }}</small>
	</div>

	<!-- Prepare authentication button -->
	<button
		v-else
		type="button"
		class="btn btn-outline-primary"
		:disabled="loading"
		@click="$emit('prepare')"
	>
		<span
			v-if="loading"
			class="spinner-border spinner-border-sm me-2"
			role="status"
			aria-hidden="true"
		></span>
		{{ $t("config.general.authPrepare") }}
	</button>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { extractDomain } from "@/utils/extractDomain";

export default defineComponent({
	name: "AuthConnectButton",
	props: {
		providerUrl: { type: String, default: undefined },
		loading: { type: Boolean, default: false },
		showHint: { type: Boolean, default: true },
	},
	emits: ["prepare"],
	computed: {
		providerDomain(): string | null {
			return this.providerUrl ? extractDomain(this.providerUrl) : null;
		},
	},
});
</script>
