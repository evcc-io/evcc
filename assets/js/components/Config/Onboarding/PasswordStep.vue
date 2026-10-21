<template>
	<form :class="{ 'was-validated': showValidation }" novalidate @submit="setPassword">
		<ErrorMessage :error="error" />
		<!-- password manager hint -->
		<input
			id="passwordSetupUsername"
			class="d-none"
			type="text"
			name="username"
			autocomplete="username"
			value="admin"
		/>
		<FormRow id="passwordSetupNew" :label="$t('passwordModal.labelNew')">
			<input
				id="passwordSetupNew"
				v-model="passwordNew"
				type="password"
				class="form-control"
				autocomplete="new-password"
				required
			/>
			<div class="invalid-feedback">{{ $t("passwordModal.empty") }}</div>
		</FormRow>
		<FormRow id="passwordSetupRepeat" :label="$t('passwordModal.labelRepeat')">
			<input
				id="passwordSetupRepeat"
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
			<button type="submit" class="btn btn-primary" :disabled="loading">
				<span
					v-if="loading"
					class="spinner-border spinner-border-sm"
					role="status"
					aria-hidden="true"
				></span>
				{{ $t("passwordModal.newPassword") }}
			</button>
		</div>
	</form>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import ErrorMessage from "../../Helper/ErrorMessage.vue";
import FormRow from "../../Helper/FormRow.vue";
import api from "@/api";
import { updateAuthStatus } from "../../Auth/auth";

export default defineComponent({
	name: "PasswordStep",
	components: { ErrorMessage, FormRow },
	emits: ["saved"],
	data() {
		return {
			passwordNew: "",
			passwordRepeat: "",
			showValidation: false,
			loading: false,
			error: "",
		};
	},
	computed: {
		passwordsMatch() {
			return this.passwordNew === this.passwordRepeat;
		},
	},
	methods: {
		async setPassword(e: Event) {
			e.preventDefault();
			const repeat = this.$refs["passwordRepeat"] as HTMLInputElement;
			repeat.setCustomValidity(this.passwordsMatch && this.passwordNew ? "" : "invalid");
			this.showValidation = true;
			if (!(e.target as HTMLFormElement).checkValidity()) return;

			this.loading = true;
			this.error = "";
			try {
				await api.put("/auth/password", { current: "", new: this.passwordNew });
				await updateAuthStatus();
				this.$emit("saved");
			} catch (error: any) {
				this.error = error.response?.data ?? error.message;
				this.showValidation = false;
			}
			this.loading = false;
		},
	},
});
</script>
