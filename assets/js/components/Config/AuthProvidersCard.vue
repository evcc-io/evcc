<template>
	<DeviceCard
		v-if="hasProviders"
		:title="$t('authProviders.title')"
		:warning="hasUnauthenticated"
		no-edit-button
		data-testid="auth-providers"
	>
		<template #icon>
			<KeyIcon />
		</template>
		<template #tags>
			<div class="auth-providers-list">
				<div
					v-for="provider in providerList"
					:key="provider.id"
					class="auth-provider-item d-flex align-items-baseline justify-content-between"
				>
					<div class="d-flex align-items-baseline">
						<span
							class="d-inline-block p-1 rounded-circle me-2"
							:class="provider.authenticated ? 'bg-success' : 'bg-warning'"
						></span>
						<span>{{ provider.title }}</span>
					</div>
					<button
						type="button"
						class="btn btn-link btn-sm p-0"
						:class="provider.authenticated ? 'text-muted' : 'text-warning'"
						@click="handleProviderAction(provider)"
					>
						{{
							provider.authenticated
								? $t("authProviders.disconnect")
								: $t("authProviders.connect")
						}}
					</button>
				</div>
			</div>
		</template>
	</DeviceCard>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import DeviceCard from "./DeviceCard.vue";
import KeyIcon from "../MaterialIcon/Key.vue";
import type { AuthProviders } from "@/types/evcc";

export interface Provider {
	title: string;
	authenticated: boolean;
	id: string;
}

export default defineComponent({
	name: "AuthProvidersCard",
	components: {
		DeviceCard,
		KeyIcon,
	},
	props: {
		providers: {
			type: Object as PropType<AuthProviders>,
			default: () => ({}),
		},
	},
	emits: ["auth-request"],
	computed: {
		hasProviders(): boolean {
			return Object.keys(this.providers).length > 0;
		},
		hasUnauthenticated(): boolean {
			return Object.values(this.providers).some((p) => !p.authenticated);
		},
		providerList(): Provider[] {
			return Object.entries(this.providers).map(([title, { authenticated, id }]) => ({
				title,
				authenticated,
				id,
			}));
		},
	},
	methods: {
		handleProviderAction(provider: Provider) {
			this.$emit("auth-request", provider.id);
		},
	},
});
</script>

<style scoped>
.auth-providers-list {
	width: 100%;
}

.auth-provider-item {
	padding: 0.5rem 0;
}
</style>
