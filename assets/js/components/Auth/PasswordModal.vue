<template>
	<GenericModal
		id="passwordUpdateModal"
		ref="modal"
		:title="$t('passwordModal.titleUpdate')"
		data-testid="password-update-modal"
		config-modal-name="passwordupdate"
		@open="open"
		@closed="closed"
	>
		<ErrorMessage :error="error" />
		<form
			v-if="modalVisible"
			ref="form"
			:class="{ 'was-validated': showValidation }"
			novalidate
			@submit="setPassword"
		>
			<!-- password manager hint -->
			<input
				id="passwordUpdateUsername"
				class="d-none"
				type="text"
				name="username"
				autocomplete="username"
				value="admin"
			/>
			<FormRow id="passwordUpdateCurrent" :label="$t('passwordModal.labelCurrent')">
				<input
					id="passwordUpdateCurrent"
					v-model="passwordCurrent"
					type="password"
					class="form-control"
					autocomplete="current-password"
				/>
			</FormRow>
			<FormRow id="passwordUpdateNew" :label="$t('passwordModal.labelNew')">
				<input
					id="passwordUpdateNew"
					v-model="passwordNew"
					type="password"
					class="form-control"
					autocomplete="new-password"
					required
				/>
				<div class="invalid-feedback">
					{{ $t("passwordModal.empty") }}
				</div>
			</FormRow>
			<FormRow id="passwordUpdateRepeat" :label="$t('passwordModal.labelRepeat')">
				<input
					id="passwordUpdateRepeat"
					ref="passwordRepeat"
					v-model="passwordRepeat"
					type="password"
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
					{{ $t("passwordModal.updatePassword") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import GenericModal from "../Helper/GenericModal.vue";
import ErrorMessage from "../Helper/ErrorMessage.vue";
import FormRow from "../Helper/FormRow.vue";
import api from "@/api";
import { updateAuthStatus } from "./auth";
import { defineComponent } from "vue";

export default defineComponent({
	name: "PasswordModal",
	components: { FormRow, GenericModal, ErrorMessage },
	data() {
		return {
			modalVisible: false,
			passwordCurrent: "",
			passwordNew: "",
			passwordRepeat: "",
			showValidation: false,
			loading: false,
			error: "" as string,
		};
	},
	computed: {
		passwordsMatch() {
			return this.passwordNew === this.passwordRepeat;
		},
		passwordEmpty() {
			return this.passwordNew.length === 0;
		},
	},
	methods: {
		open() {
			this.modalVisible = true;
		},
		closed() {
			this.modalVisible = false;
			this.passwordCurrent = "";
			this.passwordNew = "";
			this.passwordRepeat = "";
			this.error = "";
			this.loading = false;
			this.showValidation = false;
		},
		async setPassword(e: Event) {
			const passwordRepeatInput = this.$refs["passwordRepeat"] as HTMLInputElement;
			passwordRepeatInput.setCustomValidity(
				this.passwordsMatch && !this.passwordEmpty ? "" : "invalid"
			);

			e.preventDefault();
			e.stopPropagation();
			this.showValidation = true;

			const form = this.$refs["form"] as HTMLFormElement;
			if (form.checkValidity()) {
				await this.savePassword();
				await updateAuthStatus();
			}
		},
		async savePassword() {
			this.loading = true;
			this.error = "";
			try {
				const data = {
					current: this.passwordCurrent,
					new: this.passwordNew,
				};
				await api.put("/auth/password", data);
				this.loading = false;
				(this.$refs["modal"] as any)?.close();
			} catch (error: any) {
				console.error(error);
				this.error = error.response.data;
				this.showValidation = false;
			} finally {
				this.loading = false;
			}
		},
	},
});
</script>
