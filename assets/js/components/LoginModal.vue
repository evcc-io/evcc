<template>
	<GenericModal
		id="loginModal"
		:title="$t('loginModal.title')"
		size="sm"
		data-testid="login-modal"
		@open="open"
		@closed="closed"
	>
		<form v-if="modalVisible" @submit.prevent="login">
			<div class="mb-4">
				<label for="loginPassword" class="col-form-label">
					<div class="w-100">
						<span class="label">{{ $t("loginModal.password") }}</span>
					</div>
				</label>
				<input
					id="loginPassword"
					v-model="password"
					ref="password"
					class="form-control"
					autocomplete="current-password"
					type="password"
					required
				/>
			</div>

			<p v-if="error" class="text-danger my-4">{{ $t("loginModal.error") }}{{ error }}</p>
			<a
				v-if="iframeHint"
				class="text-muted my-4 d-block text-center"
				:href="evccUrl"
				target="_blank"
				data-testid="login-iframe-hint"
			>
				{{ $t("loginModal.iframeHint") }}
			</a>

			<button type="submit" class="btn btn-primary w-100 mb-3" :disabled="loading">
				<span
					v-if="loading"
					class="spinner-border spinner-border-sm"
					role="status"
					aria-hidden="true"
				></span>
				{{ $t("loginModal.login") }}
			</button>
			<a
				v-if="resetHint"
				class="text-muted my-1 d-block text-center"
				:href="resetUrl"
				target="_blank"
			>
				{{ $t("loginModal.reset") }}
			</a>
		</form>
	</GenericModal>
</template>

<script>
import GenericModal from "./GenericModal.vue";
import Modal from "bootstrap/js/dist/modal";
import api from "../api";
import { updateAuthStatus, getAndClearNextUrl, isLoggedIn } from "../auth";
import { docsPrefix } from "../i18n";

export default {
	name: "LoginModal",
	components: { GenericModal },
	data: () => {
		return {
			modalVisible: false,
			password: "",
			loading: false,
			resetHint: false,
			iframeHint: false,
			error: "",
		};
	},
	computed: {
		resetUrl() {
			return `${docsPrefix()}/docs/faq#password-reset`;
		},
		evccUrl() {
			return window.location.href;
		},
	},
	methods: {
		open() {
			this.modalVisible = true;
		},
		closed() {
			this.modalVisible = false;
			this.password = "";
			this.loading = false;
			this.error = "";
			this.resetHint = false;
			this.iframeHint = false;
		},
		focus() {
			console.log(this.$refs.password);
			this.$refs.password.focus();
		},
		closeModal() {
			Modal.getOrCreateInstance(document.getElementById("loginModal")).hide();
		},
		async login() {
			this.loading = true;

			try {
				const data = { password: this.password };
				const res = await api.post("/auth/login", data, {
					validateStatus: (code) => [200, 401].includes(code),
				});
				this.resetHint = false;
				this.iframeHint = false;
				this.error = "";
				if (res.status === 200) {
					await updateAuthStatus();
					if (isLoggedIn()) {
						this.closeModal();
						const target = getAndClearNextUrl();
						if (target) this.$router.push(target);
						this.password = "";
					} else {
						// login successful but auth cookie doesnt work
						this.error = this.$t("loginModal.iframeIssue");
						this.iframeHint = true;
					}
				}
				if (res.status === 401) {
					this.error = this.$t("loginModal.invalid");
					this.resetHint = true;
				}
			} catch (err) {
				console.error(err);
				this.error = err.message;
			} finally {
				this.loading = false;
			}
		},
	},
};
</script>
<style scoped>
form {
	/* remove body padding due to minimal content */
	margin-top: -1rem;
}
</style>
