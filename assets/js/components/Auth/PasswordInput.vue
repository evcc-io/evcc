<template>
	<div>
		<div class="mb-4">
			<label for="loginPassword" class="col-form-label">
				<div class="w-100">
					<span class="label">{{ $t("loginModal.password") }}</span>
				</div>
			</label>
			<input
				id="loginPassword"
				ref="password"
				:value="password"
				class="form-control"
				autocomplete="current-password"
				type="password"
				required
				@input="updatePassword"
			/>
		</div>

		<p v-if="error" class="text-danger my-4">{{ error }}</p>
		<a
			v-if="iframeHint"
			class="text-muted my-4 d-block text-center"
			:href="evccUrl"
			target="_blank"
			data-testid="login-iframe-hint"
		>
			{{ $t("loginModal.iframeHint") }}
		</a>
	</div>
</template>
<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "PasswordInput",
	props: {
		error: { type: String, default: "" },
		password: { type: String, default: "" },
		iframeHint: { type: Boolean, default: false },
	},
	emits: ["update:password"],
	computed: {
		evccUrl() {
			return window.location.href;
		},
	},
	methods: {
		updatePassword(e: Event) {
			this.$emit("update:password", (e.target as HTMLInputElement).value);
		},
	},
});
</script>
