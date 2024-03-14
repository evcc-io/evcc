<template>
	<Teleport to="body">
		<div
			id="setPasswordModal"
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
						<h5 class="modal-title">{{ $t("setPasswordModal.title") }}</h5>
					</div>
					<div class="modal-body">
						<form
							ref="form"
							:class="{ 'was-validated': showValidation }"
							novalidate
							@submit="submit"
						>
							<FormRow id="newPassword" :label="$t('setPasswordModal.password')">
								<input
									id="newPassword"
									type="password"
									v-model="password"
									class="form-control"
									required
								/>
								<div class="invalid-feedback">
									{{ $t("setPasswordModal.empty") }}
								</div>
							</FormRow>
							<FormRow
								id="newPasswordRepeat"
								:label="$t('setPasswordModal.passwordRepeat')"
							>
								<input
									id="newPasswordRepeat"
									type="password"
									ref="passwordRepeat"
									v-model="passwordRepeat"
									class="form-control"
								/>
								<div class="invalid-feedback">
									{{ $t("setPasswordModal.noMatch") }}
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
								>
									{{ $t("setPasswordModal.noPassword") }}
								</button>
								<button
									type="submit"
									class="btn btn-primary justify-self-right"
									@click="setPassword"
								>
									{{ $t("setPasswordModal.setPassword") }}
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
import FormRow from "./FormRow.vue";

export default {
	name: "SetPasswordModal",
	components: { FormRow },
	data: () => {
		return {
			password: "",
			passwordRepeat: "",
			showValidation: false,
		};
	},
	computed: {
		passwordsMatch: function () {
			return this.password === this.passwordRepeat;
		},
		showNoPasswordButton: function () {
			return this.showValidation && this.passwordsMatch && this.password.length === 0;
		},
	},
	methods: {
		submit: function (e) {
			console.log(this.passwordsMatch);
			this.$refs.passwordRepeat.setCustomValidity(this.passwordsMatch ? "" : "invalid");

			e.preventDefault();
			e.stopPropagation();
			this.showValidation = true;

			if (this.$refs.form.checkValidity()) {
				alert("password set");
			}
		},
	},
};
</script>
