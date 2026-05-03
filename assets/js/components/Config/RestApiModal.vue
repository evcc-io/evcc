<template>
	<GenericModal
		id="restApiModal"
		config-modal-name="restapi"
		:title="$t('config.restapi.title')"
		data-testid="restapi-modal"
	>
		<div class="container">
			<!-- Description with inline docs link -->
			<p>
				<i18n-t keypath="config.restapi.description" tag="span" scope="global">
					<template #docs>
						<a :href="docsUrl" target="_blank" rel="noopener noreferrer">
							{{ $t("config.restapi.docsLink") }}
						</a>
					</template>
				</i18n-t>
			</p>

			<!-- API base URL with copy -->
			<label class="label-sm" for="restapiUrl">
				{{ $t("config.restapi.labelUrl") }}
			</label>
			<input
				id="restapiUrl"
				type="text"
				class="form-control border font-monospace"
				:value="apiUrl"
				readonly
			/>
			<CopyLink :text="apiUrl" />

			<!-- Security alert -->
			<div
				class="alert mt-3 mb-0"
				:class="authDisabled ? 'alert-warning' : 'alert-info'"
				role="alert"
			>
				<strong>{{ $t("config.restapi.securityTitle") }}:</strong>
				{{
					authDisabled
						? $t("config.restapi.securityNoPassword")
						: $t("config.restapi.securityWithPassword")
				}}
			</div>

			<!-- Public endpoints -->
			<div class="mt-4">
				<label class="label-sm">{{ $t("config.restapi.publicTitle") }}</label>
				<p class="text-muted small mb-2">{{ $t("config.restapi.publicDescription") }}</p>
				<ul class="list-unstyled endpoint-list">
					<li v-for="endpoint in publicEndpoints" :key="endpoint" class="endpoint-row">
						<span class="badge bg-success badge-endpoint">{{
							$t("config.restapi.public")
						}}</span>
						<code>{{ endpoint }}</code>
					</li>
				</ul>
			</div>

			<!-- Protected endpoints -->
			<div class="mt-3">
				<label class="label-sm">{{ $t("config.restapi.protectedTitle") }}</label>
				<p class="text-muted small mb-2">
					{{ $t("config.restapi.protectedDescription") }}
				</p>
				<ul class="list-unstyled endpoint-list">
					<li v-for="endpoint in protectedEndpoints" :key="endpoint" class="endpoint-row">
						<span class="badge bg-warning text-dark badge-endpoint">{{
							$t("config.restapi.protected")
						}}</span>
						<code>{{ endpoint }}</code>
					</li>
				</ul>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import CopyLink from "../Helper/CopyLink.vue";
import store from "@/store";
import { docsPrefix } from "@/i18n";
import { getApiEndpoints } from "@/api";

export default defineComponent({
	name: "RestApiModal",
	components: {
		GenericModal,
		CopyLink,
	},
	data() {
		return {
			publicEndpoints: [] as string[],
			protectedEndpoints: [] as string[],
		};
	},
	computed: {
		apiUrl(): string {
			return `${window.location.protocol}//${window.location.host}/api`;
		},
		authDisabled(): boolean {
			return store.state?.authDisabled ?? false;
		},
		docsUrl(): string {
			return `${docsPrefix()}/docs/reference/api`;
		},
	},
	async mounted() {
		await this.fetchEndpoints();
	},
	methods: {
		async fetchEndpoints() {
			try {
				const endpointManifest = await getApiEndpoints();
				this.publicEndpoints = endpointManifest.public;
				this.protectedEndpoints = endpointManifest.protected;
			} catch {
				this.publicEndpoints = [];
				this.protectedEndpoints = [];
			}
		},
	},
});
</script>

<style scoped>
.container {
	padding-right: 0;
	padding-left: 0;
}

.label-sm {
	display: block;
	font-size: 0.875rem;
	margin-bottom: 0.25rem;
}

.endpoint-list {
	margin-bottom: 0;
}

.endpoint-row {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.3rem 0.5rem;
	border-radius: 4px;
}

.endpoint-row:nth-child(odd) {
	background-color: var(--bs-tertiary-bg, rgba(0, 0, 0, 0.03));
}

.badge-endpoint {
	min-width: 5.5rem;
	text-align: center;
	flex-shrink: 0;
}

code {
	font-size: 0.82rem;
}
</style>
