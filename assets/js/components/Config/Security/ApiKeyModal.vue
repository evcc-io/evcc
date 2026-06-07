<template>
	<GenericModal
		id="apiKeyModal"
		ref="modal"
		config-modal-name="apikey"
		:title="title"
		data-testid="api-key-modal"
		@open="onOpen"
		@closed="onClosed"
	>
		<div v-if="authDisabled" class="alert alert-warning">
			{{ $t("config.security.authDisabledHint") }}
		</div>

		<ErrorMessage :error="error" />

		<form v-if="view === 'overview'" @submit.prevent="submitGenerate">
			<p>{{ $t("config.apiKey.description") }}</p>

			<div class="mb-4">
				<label for="apiKeyPassword" class="col-form-label">
					<span class="label">{{ $t("loginModal.password") }}</span>
				</label>
				<input
					id="apiKeyPassword"
					v-model="password"
					class="form-control"
					autocomplete="current-password"
					type="password"
					required
					:disabled="authDisabled"
				/>
				<p v-if="passwordError" class="text-danger my-2">{{ passwordError }}</p>
			</div>

			<div class="d-flex justify-content-end">
				<button
					type="submit"
					class="btn btn-primary"
					:disabled="authDisabled || loading || !password"
				>
					<span
						v-if="loading"
						class="spinner-border spinner-border-sm me-1"
						role="status"
						aria-hidden="true"
					></span>
					<span v-if="configured">
						{{ $t("config.apiKey.regenerate") }}
					</span>
					<span v-else>
						{{ $t("config.apiKey.generate") }}
					</span>
				</button>
			</div>
		</form>

		<template v-else-if="view === 'reveal'">
			<p>{{ $t("config.apiKey.revealSuccess") }}</p>

			<FormRow id="apiKeyReveal" :label="$t('config.apiKey.keyLabel')">
				<input
					id="apiKeyReveal"
					type="text"
					class="form-control border font-monospace"
					:value="revealedKey"
					readonly
				/>
				<CopyLink :text="revealedKey || ''" />
			</FormRow>

			<FormRow id="apiKeyExample" :label="$t('config.apiKey.exampleLabel')">
				<pre
					id="apiKeyExample"
					class="form-control border font-monospace small mb-2 api-key-example"
					>{{ curlExample }}</pre
				>
				<CopyLink :text="curlExample" />
			</FormRow>

			<div class="mt-4 small text-muted">
				<strong class="text-evcc">{{ $t("general.note") }}</strong>
				{{ $t("config.apiKey.shownOnce") }}
			</div>

			<div class="d-flex justify-content-end mt-3">
				<button type="button" class="btn btn-primary" data-bs-dismiss="modal">
					{{ $t("config.general.close") }}
				</button>
			</div>
		</template>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../../Helper/GenericModal.vue";
import ErrorMessage from "../../Helper/ErrorMessage.vue";
import CopyLink from "../../Helper/CopyLink.vue";
import FormRow from "../FormRow.vue";
import api from "@/api";
import type { AxiosError } from "axios";

type View = "overview" | "reveal";

export default defineComponent({
	name: "ApiKeyModal",
	components: { GenericModal, ErrorMessage, CopyLink, FormRow },
	props: {
		authDisabled: Boolean,
	},
	data() {
		return {
			view: "overview" as View,
			configured: false,
			password: "",
			passwordError: "",
			error: "" as string | null,
			loading: false,
			revealedKey: null as string | null,
		};
	},
	computed: {
		title(): string {
			return this.view === "reveal"
				? this.$t("config.apiKey.revealTitle")
				: this.$t("config.apiKey.title");
		},
		curlExample(): string {
			const key = this.revealedKey ?? "";
			const url = `${window.location.origin}/api/system/backup`;
			return [
				`curl -X POST ${url} \\`,
				`  -H "Authorization: Bearer ${key}" \\`,
				`  -o evcc-backup.db`,
			].join("\n");
		},
	},
	methods: {
		async onOpen() {
			this.resetState();
			await this.loadStatus();
		},
		onClosed() {
			this.resetState();
		},
		resetState() {
			this.view = "overview";
			this.password = "";
			this.passwordError = "";
			this.error = "";
			this.loading = false;
			this.revealedKey = null;
		},
		async loadStatus() {
			try {
				const res = await api.get("auth/apikey");
				this.configured = !!res.data?.configured;
			} catch (err) {
				this.handleError(err);
			}
		},
		async submitGenerate() {
			if (this.authDisabled) return;
			if (this.configured && !window.confirm(this.$t("config.apiKey.regenerateWarning"))) {
				return;
			}
			this.loading = true;
			this.passwordError = "";
			this.error = "";
			try {
				const res = await api.post(
					"auth/apikey",
					{ password: this.password },
					{ validateStatus: (s) => s === 200 || s === 401 }
				);
				if (res.status === 401) {
					this.passwordError = this.$t("loginModal.invalid");
					return;
				}
				this.revealedKey = res.data?.key ?? null;
				this.configured = true;
				this.password = "";
				this.view = "reveal";
			} catch (err) {
				this.handleError(err);
			} finally {
				this.loading = false;
			}
		},
		handleError(err: unknown) {
			const axiosErr = err as AxiosError<{ error?: string } | string>;
			const data = axiosErr.response?.data;
			if (typeof data === "string") {
				this.error = data;
			} else if (data && typeof data === "object" && "error" in data) {
				this.error = (data as { error?: string }).error || axiosErr.message;
			} else {
				this.error = axiosErr.message || String(err);
			}
		},
	},
});
</script>

<style scoped>
.api-key-example {
	white-space: pre;
	overflow-x: auto;
}
</style>
