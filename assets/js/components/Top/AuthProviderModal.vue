<template>
	<GenericModal
		id="authProviderModal"
		ref="modal"
		:title="modalTitle"
		data-testid="auth-provider-modal"
		@open="handleOpen"
		@closed="handleClosed"
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

				<!-- Challenge (server-side login) -->
				<AuthChallenge
					v-if="auth.challenge"
					id="authProviderChallenge"
					v-model="challengeAnswer"
					:challenge="auth.challenge"
					@submit="submitChallenge"
				/>

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
					<button
						v-if="auth.challenge"
						type="button"
						class="btn btn-primary"
						:disabled="auth.loading || !challengeAnswer"
						@click="submitChallenge"
					>
						<span
							v-if="auth.loading"
							class="spinner-border spinner-border-sm me-2"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t("authProviders.challenge.submit") }}
					</button>
					<AuthConnectButton
						v-else
						:provider-url="auth.providerUrl ?? undefined"
						:loading="auth.loading"
						@prepare="prepareAuthentication"
						@external-click="waitingForAuthentication = true"
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
import GenericModal from "../Helper/GenericModal.vue";
import AuthCodeDisplay from "../Config/AuthCodeDisplay.vue";
import AuthChallenge from "../Config/AuthChallenge.vue";
import AuthConnectButton from "../Config/AuthConnectButton.vue";
import {
	initialAuthState,
	prepareAuthLogin,
	submitAuthChallenge,
	performAuthLogout,
} from "../Config/utils/authProvider";
import type { Provider } from "./types";

export default defineComponent({
	name: "AuthProviderModal",
	components: {
		GenericModal,
		AuthCodeDisplay,
		AuthChallenge,
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
			challengeAnswer: "",
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
			// auth.ok: login completed here, the websocket confirms shortly after
			return this.isAuthenticated && (this.waitingForAuthentication || this.auth.ok);
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
		"auth.challenge"() {
			// a wrong answer comes back as a fresh challenge
			this.challengeAnswer = "";
		},
		challengeAnswer() {
			// outdated errors must not persist while typing
			this.auth.error = null;
		},
	},
	methods: {
		// on open rather than on provider change: reopening for the same provider
		// would otherwise show an empty form
		handleOpen() {
			this.reset();
			// auto-run the prepare step. no user input needed
			this.prepareAuthentication();
		},
		reset() {
			this.auth = initialAuthState();
			this.logoutLoading = false;
			this.logoutError = null;
			this.waitingForAuthentication = false;
		},
		handleClosed() {
			this.reset();
		},
		async prepareAuthentication() {
			if (!this.providerId || this.isAuthenticated) return;
			await prepareAuthLogin(this.auth, this.providerId);
		},
		async submitChallenge() {
			if (!this.challengeAnswer) return;
			await submitAuthChallenge(this.auth, this.challengeAnswer);
		},
		async performLogout() {
			if (!this.providerId) return;

			this.logoutLoading = true;
			this.logoutError = null;

			const result = await performAuthLogout(this.providerId);
			if (result.success) {
				(this.$refs["modal"] as any)?.close();
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
