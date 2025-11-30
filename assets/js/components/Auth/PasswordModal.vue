<template>
	<GenericModal
		:id="modalId"
		ref="modal"
		:title="title"
		:data-testid="dataTestId"
		:uncloseable="isUncloseable"
		@open="open"
		@closed="closed"
	>
		<p v-if="error" class="text-danger">{{ $t("passwordModal.error") }} "{{ error }}"</p>
		<p v-if="isSetupMode" class="mb-4">{{ $t("passwordModal.description") }}</p>
		<form
			v-if="modalVisible"
			ref="form"
			:class="{ 'was-validated': showValidation }"
			novalidate
			@submit="setPassword"
		>
			<!-- password manager hint -->
			<input
				:id="formId('Username')"
				class="d-none"
				type="text"
				name="username"
				autocomplete="username"
				value="admin"
			/>
			<FormRow
				v-if="updateMode"
				:id="formId('Current')"
				:label="$t('passwordModal.labelCurrent')"
			>
				<input
					:id="formId('Current')"
					v-model="passwordCurrent"
					type="password"
					class="form-control"
					autocomplete="current-password"
				/>
			</FormRow>
			<FormRow :id="formId('New')" :label="$t('passwordModal.labelNew')">
				<input
					:id="formId('New')"
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
			<FormRow :id="formId('Repeat')" :label="$t('passwordModal.labelRepeat')">
				<input
					:id="formId('Repeat')"
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

<script lang="ts">
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "../Helper/FormRow.vue";
import api from "@/api";
import { updateAuthStatus } from "./auth";
import { defineComponent } from "vue";

export default defineComponent({
	name: "PasswordModal",
	components: { FormRow, GenericModal },
	props: {
		updateMode: Boolean,
	},
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
		modalId() {
			return this.updateMode ? "passwordUpdateModal" : "passwordSetupModal";
		},
		dataTestId() {
			return this.updateMode ? "password-update-modal" : "password-setup-modal";
		},
		isSetupMode() {
			return !this.updateMode;
		},
		isUncloseable() {
			return !this.updateMode;
		},
		passwordsMatch() {
			return this.passwordNew === this.passwordRepeat;
		},
		passwordEmpty() {
			return this.passwordNew.length === 0;
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
		formId(name: string) {
			const prefix = this.updateMode ? "passwordUpdate" : "passwordSetup";
			return `${prefix}_${name}`;
		},
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
