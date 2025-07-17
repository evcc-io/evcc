<template>
	<div>
		<div class="mb-4">
			<label for="loginPassword" class="col-form-label">
				<div class="w-100">
					<span class="label">{{ labelText }}</span>
				</div>
			</label>
			<input
				id="loginPassword"
				ref="password"
				:value="password"
				class="form-control"
				autocomplete="current-password"
				type="password"
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
		label: { type: String },
	},
	emits: ["update:password"],
	computed: {
		evccUrl() {
			return window.location.href;
		},
		labelText() {
			return this.label || this.$t("loginModal.password");
		},
	},
	methods: {
		updatePassword(e: Event) {
			this.$emit("update:password", (e.target as HTMLInputElement).value);
		},
	},
});
</script>
