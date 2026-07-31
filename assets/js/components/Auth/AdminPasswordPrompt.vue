<template>
	<div class="alert alert-warning mt-4 pb-0" role="alert" data-testid="admin-password-prompt">
		<p class="fw-bold mb-2">{{ $t("config.adminPassword.title") }}</p>
		<p>{{ $t("config.adminPassword.description") }}</p>
		<PasswordInput
			class="text-body"
			:password="password"
			:error="invalid ? $t('config.adminPassword.invalid') : ''"
			@update:password="$emit('update:password', $event)"
		/>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import PasswordInput from "./PasswordInput.vue";

export default defineComponent({
	name: "AdminPasswordPrompt",
	components: { PasswordInput },
	props: {
		password: { type: String, default: "" },
		invalid: Boolean,
	},
	emits: ["update:password"],
});
</script>
