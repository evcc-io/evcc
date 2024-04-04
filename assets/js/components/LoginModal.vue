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
			<p v-if="error" class="text-danger">{{ $t("loginModal.error") }}{{ error }}</p>

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

			<button type="submit" class="btn btn-primary w-100 mb-3" :disabled="loading">
				{{ $t("loginModal.login") }}
			</button>
		</form>
	</GenericModal>
</template>

<script>
import GenericModal from "./GenericModal.vue";
import Modal from "bootstrap/js/dist/modal";
import api from "../api";
import { updateAuthStatus } from "../auth";

export default {
	name: "LoginModal",
	components: { GenericModal },
	data: () => {
		return {
			modalVisible: false,
			password: "",
			loading: false,
			error: "",
		};
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
			this.error = "";

			try {
				const data = { password: this.password };
				const res = await api.post("/auth/login", data, {
					validateStatus: (code) => [200, 401].includes(code),
				});
				if (res.status === 200) {
					this.closeModal();
					await updateAuthStatus();
					this.$router.push({ path: "/config" });
					this.password = "";
				}
				if (res.status === 401) {
					this.error = this.$t("loginModal.invalid");
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
