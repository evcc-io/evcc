<template>
	<GenericModal
		id="loginModal"
		:title="$t('loginModal.title')"
		:size="modalSize"
		data-testid="login-modal"
		@open="open"
		@closed="closed"
	>
		<div v-if="demoMode" class="alert alert-warning" role="alert">
			{{ $t("loginModal.demoMode") }}
		</div>
		<form v-else-if="modalVisible" @submit.prevent="login">
			<PasswordInput v-model:password="password" :error="error" :iframe-hint="iframeHint" />

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

<script lang="ts">
import GenericModal from "../Helper/GenericModal.vue";
import Modal from "bootstrap/js/dist/modal";
import api from "@/api";
import { updateAuthStatus, getAndClearNextUrl, getAndClearNextModal, isLoggedIn } from "./auth";
import { docsPrefix } from "@/i18n";
import { defineComponent } from "vue";
import PasswordInput from "./PasswordInput.vue";

export default defineComponent({
	name: "LoginModal",
	components: { GenericModal, PasswordInput },
	props: {
		demoMode: Boolean,
	},
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
		modalSize() {
			return this.demoMode ? "md" : "sm";
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
			console.log(this.$refs["password"]);
			this.$refs["password"]?.focus();
		},
		closeModal() {
			Modal.getOrCreateInstance(document.getElementById("loginModal") as HTMLElement).hide();
		},
		async login() {
			this.loading = true;

			try {
				const data = { password: this.password };
				const res = await api.post("/auth/login", data, {
					validateStatus: (code) => [200, 401, 403].includes(code),
				});
				this.resetHint = false;
				this.iframeHint = false;
				this.error = "";
				if (res.status === 200) {
					await updateAuthStatus();
					if (isLoggedIn()) {
						this.closeModal();
						const url = getAndClearNextUrl();
						if (url) this.$router.push(url);
						const modal = getAndClearNextModal();
						if (modal) modal.show();
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
				if (res.status === 403) {
					this.error = this.$t("loginModal.demoMode");
				}
			} catch (err) {
				console.error(err);
				if (err instanceof Error) {
					this.error = err.message;
				}
			} finally {
				this.loading = false;
			}
		},
	},
});
</script>
<style scoped>
form {
	/* remove body padding due to minimal content */
	margin-top: -1rem;
}
</style>
