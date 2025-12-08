<template>
	<GenericModal
		id="authProviderModal"
		ref="modal"
		:title="modalTitle"
		data-testid="auth-provider-modal"
		@close="handleClose"
	>
		<div class="container mx-0 px-0">
			<!-- Success message after authentication -->
			<template v-if="showAuthenticationSuccess">
				<p class="mb-4 text-success">
					{{ $t("authProviders.success", { title: providerTitle }) }}<br />
					{{ $t("authProviders.successCloseModal") }}
				</p>
				<div class="d-flex justify-content-end">
					<button type="button" class="btn btn-primary" data-bs-dismiss="modal">
						{{ $t("config.general.close") }}
					</button>
				</div>
			</template>

			<!-- Login flow -->
			<template v-else-if="showAuthentication">
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
					<AuthCodeDisplay
						id="authProviderCode"
						:code="auth.code"
						:expiry="auth.expiry"
					/>
				</div>

				<!-- Error display -->
				<p v-if="auth.error" class="text-danger mt-3">{{ auth.error }}</p>

				<!-- Action buttons -->
				<div
					class="my-4 d-flex align-items-stretch justify-content-sm-between align-items-sm-baseline flex-column-reverse flex-sm-row gap-2"
				>
					<button type="button" class="btn btn-link text-muted" data-bs-dismiss="modal">
						{{ $t("config.general.cancel") }}
					</button>

					<!-- Authentication buttons -->
					<AuthConnectButton
						:provider-url="auth.providerUrl ?? undefined"
						:loading="auth.loading"
						@prepare="prepareAuthentication"
					/>
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
				<div
					class="my-4 d-flex align-items-stretch justify-content-sm-between align-items-sm-baseline flex-column-reverse flex-sm-row gap-2"
				>
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
import AuthCodeDisplay from "../Config/AuthCodeDisplay.vue";
import AuthConnectButton from "../Config/AuthConnectButton.vue";
import {
	initialAuthState,
	prepareAuthLogin,
	performAuthLogout,
} from "../Config/utils/authProvider";
import type { Provider } from "./types";

export default defineComponent({
	name: "AuthProviderModal",
	components: {
		GenericModal,
		AuthCodeDisplay,
		AuthConnectButton,
	},
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
			waitingForAuthentication: false,
		};
	},
	computed: {
		isAuthenticated(): boolean {
			return this.provider?.authenticated || false;
		},
		showAuthentication(): boolean {
			return !this.isAuthenticated;
		},
		showAuthenticationSuccess(): boolean {
			return this.isAuthenticated && this.waitingForAuthentication;
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
			this.waitingForAuthentication = false;
		},
		handleClose() {
			this.reset();
		},
		async prepareAuthentication() {
			if (!this.providerId) return;
			this.waitingForAuthentication = true;
			await prepareAuthLogin(this.auth, this.providerId);
		},
		async performLogout() {
			if (!this.providerId) return;

			this.logoutLoading = true;
			this.logoutError = null;

			const result = await performAuthLogout(this.providerId);
			if (result.success) {
				const modalElement = (this.$refs["modal"] as any)?.$el;
				if (modalElement) {
					Modal.getInstance(modalElement)?.hide();
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
