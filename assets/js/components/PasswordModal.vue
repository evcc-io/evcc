<template>
	<Teleport to="body">
		<div
			id="passwordModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
			data-bs-backdrop="static"
			data-bs-keyboard="false"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("passwordModal.title") }}</h5>
					</div>
					<div class="modal-body">
						<p v-if="error" class="text-danger">
							{{ $t("passwordModal.error") }} "{{ error }}"
						</p>
						<p class="mb-4">{{ $t("passwordModal.description") }}</p>
						<form
							ref="form"
							:class="{ 'was-validated': showValidation }"
							novalidate
							@submit="setPassword"
						>
							<!-- password manager hint -->
							<input
								class="d-none"
								type="text"
								name="username"
								id="username"
								autocomplete="username"
								value="admin"
							/>
							<FormRow id="newPassword" :label="$t('passwordModal.password')">
								<input
									id="newPassword"
									type="password"
									v-model="password"
									class="form-control"
									autocomplete="new-password"
									required
								/>
								<div class="invalid-feedback">
									{{ $t("passwordModal.empty") }}
								</div>
							</FormRow>
							<FormRow
								id="newPasswordRepeat"
								:label="$t('passwordModal.passwordRepeat')"
							>
								<input
									id="newPasswordRepeat"
									type="password"
									ref="passwordRepeat"
									v-model="passwordRepeat"
									class="form-control"
									autocomplete="new-password"
								/>
								<div v-if="!passwordsMatch" class="invalid-feedback">
									{{ $t("passwordModal.noMatch") }}
								</div>
							</FormRow>

							<div class="d-flex justify-content-end">
								<button
									type="submit"
									class="btn btn-primary justify-self-right"
									:disabled="loading"
								>
									<span
										v-if="loading"
										class="spinner-border spinner-border-sm"
										role="status"
										aria-hidden="true"
									></span>
									{{ $t("passwordModal.setPassword") }}
								</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import FormRow from "./FormRow.vue";
import api from "../api";
import { updateAuthStatus } from "../auth";

export default {
	name: "PasswordModal",
	components: { FormRow },
	data: () => {
		return {
			password: "",
			passwordRepeat: "",
			showValidation: false,
			loading: false,
			error: false,
		};
	},
	computed: {
		passwordsMatch: function () {
			return this.password === this.passwordRepeat;
		},
		passwordEmpty: function () {
			return this.password.length === 0;
		},
	},
	methods: {
		setPassword: async function (e) {
			this.$refs.passwordRepeat.setCustomValidity(
				this.passwordsMatch && !this.passwordEmpty ? "" : "invalid"
			);

			e.preventDefault();
			e.stopPropagation();
			this.showValidation = true;

			if (this.$refs.form.checkValidity()) {
				await this.savePassword(this.password);
				await updateAuthStatus();
				// reset form
				this.password = "";
				this.passwordRepeat = "";
			}
		},
		savePassword: async function (password) {
			this.loading = true;
			this.error = null;
			try {
				await api.post("/password", password);
			} catch (error) {
				console.error(error);
				this.error = error.response.data;
			} finally {
				this.loading = false;
			}
		},
		openPasswordModal() {
			Modal.getOrCreateInstance(document.getElementById("passwordModal")).show();
		},
	},
};
</script>
