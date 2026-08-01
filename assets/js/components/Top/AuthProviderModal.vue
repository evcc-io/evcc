<template>
	<GenericModal
		id="authProviderModal"
		ref="modal"
		:title="modalTitle"
		data-testid="auth-provider-modal"
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
				<!-- Interactive (credential) providers, e.g. a captcha login -->
				<template v-if="isInteractive">
					<p class="mb-3">{{ interactivePrompt }}</p>

					<img
						v-if="challenge?.image"
						:src="challenge.image"
						alt=""
						class="challenge-img mb-3"
					/>

					<input
						v-for="field in challengeFields"
						:key="field.name"
						v-model="values[field.name]"
						:type="inputType(field.type)"
						class="form-control mb-2"
						autocomplete="off"
						autocapitalize="off"
						:placeholder="fieldLabel(field.name)"
						@keyup.enter="submitChallenge"
					/>

					<p v-if="interactiveError" class="text-danger mt-2">{{ interactiveError }}</p>

					<div
						class="my-4 d-flex justify-content-sm-between flex-column-reverse flex-sm-row gap-2"
					>
						<button
							type="button"
							class="btn btn-link text-muted"
							data-bs-dismiss="modal"
						>
							{{ $t("config.general.cancel") }}
						</button>
						<button
							type="button"
							class="btn btn-primary"
							:disabled="submitting || !challenge"
							@click="submitChallenge"
						>
							<span
								v-if="submitting"
								class="spinner-border spinner-border-sm me-2"
								role="status"
								aria-hidden="true"
							></span>
							{{ $t("authProviders.interactiveSubmit") }}
						</button>
					</div>
				</template>

				<p v-if="!isInteractive" class="mb-4">
					{{
						$t("authProviders.modalDescriptionLogin", {
							provider: providerTitle,
						})
					}}
				</p>

				<!-- Auth code display (device flow) -->
				<div v-if="!isInteractive && auth.code">
					<hr class="my-4" />
					<AuthCodeDisplay
						id="authProviderCode"
						:code="auth.code"
						:expiry="auth.expiry"
					/>
				</div>

				<!-- Error display -->
				<p v-if="!isInteractive && auth.error" class="text-danger mt-3">{{ auth.error }}</p>

				<!-- Action buttons -->
				<div
					v-if="!isInteractive"
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
import restart from "@/restart";
import GenericModal from "../Helper/GenericModal.vue";
import AuthCodeDisplay from "../Config/AuthCodeDisplay.vue";
import AuthConnectButton from "../Config/AuthConnectButton.vue";
import {
	initialAuthState,
	prepareAuthLogin,
	performAuthLogout,
	fetchAuthChallenge,
	submitAuthChallenge,
} from "../Config/utils/authProvider";
import type { Provider } from "./types";
import type { AuthChallenge } from "@/types/evcc";

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
			// interactive (credential) login state
			challenge: null as AuthChallenge | null,
			values: {} as Record<string, string>,
			submitting: false,
			interactiveError: null as string | null,
		};
	},
	computed: {
		isAuthenticated(): boolean {
			return this.provider?.authenticated || false;
		},
		isInteractive(): boolean {
			return this.provider?.interactive || false;
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
		challengeFields() {
			return this.challenge?.fields || [];
		},
		interactivePrompt(): string {
			if (this.challenge?.image) {
				return this.$t("authProviders.captchaHint");
			}
			return this.$t("authProviders.modalDescriptionLogin", {
				provider: this.providerTitle,
			});
		},
	},
	watch: {
		providerId(newId) {
			if (newId) {
				this.reset();
				if (this.isInteractive) {
					this.loadChallenge();
				} else {
					// auto-run the prepare step. no user input needed
					this.prepareAuthentication();
				}
			}
		},
	},
	methods: {
		reset() {
			this.auth = initialAuthState();
			this.logoutLoading = false;
			this.logoutError = null;
			this.waitingForAuthentication = false;
			this.challenge = null;
			this.values = {};
			this.submitting = false;
			this.interactiveError = null;
		},
		inputType(type: string): string {
			return ["email", "password", "text"].includes(type) ? type : "text";
		},
		fieldLabel(name: string): string {
			const key = `authProviders.field.${name}`;
			return this.$te(key) ? this.$t(key) : name;
		},
		async loadChallenge() {
			this.interactiveError = null;
			this.challenge = null;
			this.values = {};
			try {
				const challenge = await fetchAuthChallenge(this.providerId);
				if (challenge) {
					this.challenge = challenge;
				} else {
					// no interactive form to show; nothing more to do here
					this.interactiveError = this.$t("authProviders.loginFailed");
				}
			} catch (e: any) {
				this.interactiveError = e.message || this.$t("authProviders.loginFailed");
			}
		},
		async submitChallenge() {
			if (this.submitting || !this.challenge) return;
			this.interactiveError = null;
			this.submitting = true;
			try {
				const result = await submitAuthChallenge(this.providerId, this.values);
				if (result.error) {
					this.interactiveError = result.error;
				} else if (result.challenge) {
					// next step (e.g. a captcha)
					this.challenge = result.challenge;
					this.values = {};
				} else if (result.authenticated) {
					this.challenge = null;
					this.waitingForAuthentication = true;
					// devices using the provider are reinitialized on restart
					restart.restartNeeded = true;
				}
			} catch (e: any) {
				this.interactiveError = e.message || this.$t("authProviders.loginFailed");
			} finally {
				this.submitting = false;
			}
		},
		handleClosed() {
			this.reset();
		},
		async prepareAuthentication() {
			if (!this.providerId || this.isAuthenticated) return;
			await prepareAuthLogin(this.auth, this.providerId);
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
.challenge-img {
	display: block;
	max-width: 100%;
	height: auto;
	border: 1px solid var(--bs-border-color);
	border-radius: var(--bs-border-radius);
	background: #fff;
}
</style>
