<template>
	<GenericModal
		id="authProviderModal"
		ref="modal"
		:title="modalTitle"
		data-testid="auth-provider-modal"
		@close="handleClose"
	>
		<div class="container mx-0 px-0">
			<!-- Login flow -->
			<template v-if="!isAuthenticated">
				<p class="mb-4">
					{{
						$t("authProviders.modalDescriptionLogin", {
							provider: providerTitle,
						})
					}}
				</p>

				<!-- Auth code display (device flow) -->
				<div v-if="auth.code">
					<hr class="my-4" />
					<FormRow
						id="authProviderCode"
						:label="$t('authProviders.authCode')"
						:help="
							$t('authProviders.authCodeHelp', {
								duration: codeValidityDuration,
							})
						"
					>
						<input
							id="authProviderCode"
							type="text"
							class="form-control fs-2 border font-monospace"
							:value="auth.code"
							readonly
						/>
						<CopyLink :text="auth.code" />
					</FormRow>
				</div>

				<!-- Error display -->
				<p v-if="auth.error" class="text-danger mt-3">{{ auth.error }}</p>

				<!-- Action buttons -->
				<div class="mt-4 d-flex justify-content-between align-items-center">
					<button type="button" class="btn btn-link text-muted" data-bs-dismiss="modal">
						{{ $t("config.general.cancel") }}
					</button>

					<!-- Connect to provider button (when URL is available) -->
					<div v-if="auth.providerUrl" class="d-flex flex-column align-items-end gap-2">
						<a :href="auth.providerUrl" target="_blank" class="btn btn-primary">
							{{
								$t("authProviders.buttonConnect", {
									provider: providerDomain,
								})
							}}
						</a>
						<small class="d-block">{{ $t("config.general.authPerformHint") }}</small>
					</div>

					<!-- Prepare authentication button -->
					<button
						v-else
						type="button"
						class="btn btn-outline-primary"
						:disabled="auth.loading"
						@click="prepareAuthentication"
					>
						<span
							v-if="auth.loading"
							class="spinner-border spinner-border-sm me-2"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t("config.general.authPrepare") }}
					</button>
				</div>
			</template>

			<!-- Logout flow -->
			<template v-else>
				<p class="mb-4">
					{{
						$t("authProviders.modalDescriptionLogout", {
							provider: providerTitle,
						})
					}}
				</p>

				<!-- Error display -->
				<p v-if="logoutError" class="text-danger mt-3">{{ logoutError }}</p>

				<!-- Action buttons -->
				<div class="mt-4 d-flex justify-content-between align-items-center">
					<button type="button" class="btn btn-link text-muted" data-bs-dismiss="modal">
						{{ $t("config.general.cancel") }}
					</button>

					<button
						type="button"
						class="btn btn-danger"
						:disabled="logoutLoading"
						@click="performLogout"
					>
						<span
							v-if="logoutLoading"
							class="spinner-border spinner-border-sm me-2"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t("authProviders.buttonDisconnect") }}
					</button>
				</div>
			</template>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Modal from "bootstrap/js/dist/modal";
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "../Config/FormRow.vue";
import CopyLink from "../Helper/CopyLink.vue";
import {
	initialAuthState,
	performAuthLogin,
	performAuthLogout,
} from "../Config/utils/authProvider";
import { extractDomain } from "@/utils/extractDomain";
import formatter from "@/mixins/formatter";
import type { Provider } from "./types";

export default defineComponent({
	name: "AuthProviderModal",
	components: {
		GenericModal,
		FormRow,
		CopyLink,
	},
	mixins: [formatter],
	props: {
		provider: {
			type: Object as PropType<Provider | null>,
			default: null,
		},
	},
	data() {
		return {
			logoutLoading: false,
			logoutError: null as string | null,
			auth: initialAuthState(),
		};
	},
	computed: {
		isAuthenticated(): boolean {
			return this.provider?.authenticated || false;
		},
		modalTitle(): string {
			return this.providerTitle;
		},
		providerTitle(): string {
			return this.provider?.title || "Unknown";
		},
		providerId(): string {
			return this.provider?.id || "";
		},
		providerDomain(): string | null {
			return this.auth.providerUrl ? extractDomain(this.auth.providerUrl) : null;
		},
		codeValidityDuration(): string | null {
			if (!this.auth.expiry) return null;
			const seconds = Math.max(
				0,
				Math.floor((this.auth.expiry.getTime() - new Date().getTime()) / 1000)
			);
			return this.fmtDurationLong(seconds);
		},
	},
	watch: {
		provider(newProvider) {
			if (newProvider) {
				this.reset();
			}
		},
	},
	methods: {
		reset() {
			this.auth = initialAuthState();
			this.logoutLoading = false;
			this.logoutError = null;
		},
		handleClose() {
			this.reset();
		},
		async prepareAuthentication() {
			if (!this.providerId) return;

			const result = await performAuthLogin(this.auth, this.providerId);
			if (!result.success) {
				console.error("Authentication preparation failed:", result.error);
			}
		},
		async performLogout() {
			if (!this.providerId) return;

			this.logoutLoading = true;
			this.logoutError = null;

			const result = await performAuthLogout(this.providerId);
			if (result.success) {
				// Close the modal on successful logout
				const modalElement = document.getElementById("authProviderModal");
				if (modalElement) {
					const modal = Modal.getInstance(modalElement);
					modal?.hide();
				}
			} else {
				this.logoutError = result.error || this.$t("authProviders.logoutFailed");
			}
			this.logoutLoading = false;
		},
	},
});
</script>

<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
</style>
