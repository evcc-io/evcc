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
							<FormRow id="newPassword" :label="$t('passwordModal.password')">
								<input
									id="newPassword"
									type="password"
									v-model="password"
									class="form-control"
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
								/>
								<div v-if="!passwordsMatch" class="invalid-feedback">
									{{ $t("passwordModal.noMatch") }}
								</div>
							</FormRow>

							<div
								class="d-flex"
								:class="
									showNoPasswordButton
										? 'justify-content-between'
										: 'justify-content-end'
								"
							>
								<button
									v-if="showNoPasswordButton"
									type="button"
									class="btn btn-link text-danger"
									data-bs-dismiss="modal"
									:disabled="loading"
									@click="openNoPasswordModal"
								>
									{{ $t("passwordModal.noPassword") }}
								</button>
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

	<Teleport to="body">
		<div
			id="noPasswordModal"
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
						<h5>{{ $t("passwordModal.noPasswordReally") }}</h5>
					</div>
					<div class="modal-footer d-flex justify-content-between">
						<p v-if="error" class="text-danger mb-4">
							{{ $t("passwordModal.error") }} "{{ error }}"
						</p>
						<button
							type="button"
							class="btn btn-outline-secondary"
							data-bs-dismiss="modal"
							@click="openPasswordModal"
						>
							{{ $t("passwordModal.noPasswordCancel") }}
						</button>
						<button type="button" class="btn btn-danger" @click="setNoPassword">
							{{ $t("passwordModal.noPasswordConfirm") }}
						</button>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import FormRow from "./FormRow.vue";
import Modal from "bootstrap/js/dist/modal";
import api from "../api";

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
		showNoPasswordButton: function () {
			return this.showValidation && this.passwordsMatch && this.passwordEmpty;
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
				const success = await this.savePassword(this.password);
				if (success) {
					this.closePasswordModal();
				}
			}
		},
		savePassword: async function (password) {
			this.loading = true;
			this.error = null;
			try {
				await api.post("/password", { password });
				return true;
			} catch (error) {
				console.error(error);
				this.error = error.response.data;
			} finally {
				this.loading = false;
			}
			return false;
		},
		setNoPassword: async function () {
			const success = await this.savePassword("");
			if (success) {
				this.closeNoPasswordModal();
			}
		},
		closePasswordModal() {
			const modal = Modal.getInstance(document.getElementById("passwordModal"));
			modal.hide();
		},
		closeNoPasswordModal() {
			const modal = Modal.getInstance(document.getElementById("noPasswordModal"));
			modal.hide();
		},
		openPasswordModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("passwordModal"));
			modal.show();
		},
		openNoPasswordModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("noPasswordModal"));
			modal.show();
		},
	},
};
</script>
