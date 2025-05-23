<template>
	<GenericModal
		id="passwordModal"
		ref="modal"
		:title="title"
		data-testid="password-modal"
		:uncloseable="!updateMode"
		@open="open"
		@closed="closed"
	>
		<p v-if="error" class="text-danger">{{ $t("passwordModal.error") }} "{{ error.trim() }}"</p>
		<p v-if="!updateMode" class="mb-4">{{ $t("passwordModal.description") }}</p>
		<form
			v-if="modalVisible"
			ref="form"
			:class="{ 'was-validated': showValidation }"
			novalidate
			@submit="setPassword"
		>
			<!-- password manager hint -->
			<input
				id="username"
				class="d-none"
				type="text"
				name="username"
				autocomplete="username"
				value="admin"
			/>
			<FormRow
				v-if="updateMode"
				id="passwordCurrent"
				:label="$t('passwordModal.labelCurrent')"
			>
				<input
					id="passwordCurrent"
					v-model="passwordCurrent"
					type="password"
					class="form-control"
					autocomplete="current-password"
				/>
			</FormRow>
			<FormRow id="passwordNew" :label="$t('passwordModal.labelNew')">
				<input
					id="passwordNew"
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
			<FormRow id="passwordRepeat" :label="$t('passwordModal.labelRepeat')">
				<input
					id="passwordRepeat"
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
					{{ submitText }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script type="ts">
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "../Helper/FormRow.vue";
import api from "@/api";
import { updateAuthStatus, isConfigured } from "./auth";
import  { defineComponent } from "vue";

export default defineComponent({
	name: "PasswordModal",
	components: { FormRow, GenericModal },
	data: () => {
		return {
			modalVisible: false,
			passwordCurrent: "",
			passwordNew: "",
			passwordRepeat: "",
			showValidation: false,
			loading: false,
			error: false,
		};
	},
	computed: {
		passwordsMatch() {
			return this.passwordNew === this.passwordRepeat;
		},
		passwordEmpty() {
			return this.passwordNew.length === 0;
		},
		updateMode() {
			return isConfigured();
		},
		title() {
			return this.updateMode
				? this.$t("passwordModal.titleUpdate")
				: this.$t("passwordModal.titleNew");
		},
		submitText() {
			return this.updateMode
				? this.$t("passwordModal.updatePassword")
				: this.$t("passwordModal.newPassword");
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
			this.error = false;
			this.loading = false;
			this.showValidation = false;
		},
		async setPassword(e) {
			this.$refs.passwordRepeat.setCustomValidity(
				this.passwordsMatch && !this.passwordEmpty ? "" : "invalid"
			);

			e.preventDefault();
			e.stopPropagation();
			this.showValidation = true;

			if (this.$refs.form.checkValidity()) {
				await this.savePassword();
				await updateAuthStatus();
			}
		},
		async savePassword() {
			this.loading = true;
			this.error = null;
			try {
				const data = {
					current: this.passwordCurrent,
					new: this.passwordNew,
				};
				await api.put("/auth/password", data);
				this.loading = false;
				this.$refs.modal?.close();
			} catch (error) {
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
