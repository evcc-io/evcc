<template>
	<Teleport to="body">
		<div
			id="loginModal"
			class="modal fade text-dark modal-sm"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="false"
			data-testid="login-modal"
		>
			<div class="modal-dialog modal-dialog-centered open" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("loginModal.title") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<form @submit.prevent="login">
							<p v-if="error" class="text-danger">
								{{ $t("loginModal.error") }}{{ error }}
							</p>

							<div class="mb-4">
								<label for="loginPassword" class="col-form-label">
									<div class="w-100">
										<span class="label">{{ $t("loginModal.password") }}</span>
									</div>
								</label>
								<input
									id="loginPassword"
									v-model="password"
									class="form-control"
									autocomplete="current-password"
									type="password"
									required
								/>
							</div>

							<button type="submit" class="btn btn-primary w-100" :disabled="loading">
								{{ $t("loginModal.login") }}
							</button>
						</form>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import api from "../api";
import { updateAuthStatus } from "../auth";

export default {
	name: "LoginModal",
	data: () => {
		return {
			password: "",
			loading: false,
			error: "",
		};
	},
	methods: {
		closeModal() {
			Modal.getOrCreateInstance(document.getElementById("loginModal")).hide();
		},
		async login() {
			this.loading = true;
			this.error = "";

			try {
				const res = await api.post("/auth/login", this.password, {
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
