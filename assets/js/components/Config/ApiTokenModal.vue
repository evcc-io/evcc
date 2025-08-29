<template>
	<GenericModal
		id="apiTokenModal"
		ref="modal"
		:title="$t('config.apitoken.title')"
		data-testid="api-token-modal"
		@open="open"
		@closed="closed"
	>
		<div v-if="!confirm && !token">
			<p class="mb-4">
				{{ $t("config.apitoken.description") }}
				<a :href="docsUrl" target="_blank" rel="noopener noreferrer" class="text-primary">
					{{ $t("config.apitoken.docsLink") }}
				</a>
			</p>

			<div class="d-flex justify-content-end">
				<button type="button" class="btn btn-primary" @click="showConfirmStep">
					{{ $t("config.apitoken.generateButton") }}
				</button>
			</div>
		</div>

		<form v-else-if="confirm && !token" @submit.prevent="generateToken">
			<p>
				{{ $t("config.apitoken.description") }}
				<a :href="docsUrl" target="_blank" rel="noopener noreferrer" class="text-primary">
					{{ $t("config.apitoken.docsLink") }}
				</a>
			</p>

			<PasswordInput ref="passwordInput" v-model:password="password" :error="error" />

			<div class="d-flex justify-content-between gap-2 flex-wrap">
				<button
					type="button"
					class="btn btn-outline-secondary"
					:disabled="loading"
					data-bs-dismiss="modal"
					@click="cancelConfirm"
				>
					{{ $t("config.apitoken.cancel") }}
				</button>
				<button type="submit" class="btn btn-primary" :disabled="loading">
					<span
						v-if="loading"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.apitoken.confirmButton") }}
				</button>
			</div>
		</form>

		<div v-else-if="token">
			<p class="mb-4">
				{{ $t("config.apitoken.tokenGenerated") }}
			</p>

			<div class="mb-3">
				<label for="apiTokenOutput" class="form-label">
					{{ $t("config.apitoken.tokenLabel") }}
				</label>
				<textarea
					id="apiTokenOutput"
					v-model="token"
					class="form-control font-monospace"
					rows="6"
					readonly
					@focus="selectToken"
				/>
				<CopyLink :text="token" />
			</div>

			<div class="mb-3">
				<label for="apiTokenCurl" class="form-label">
					{{ $t("config.apitoken.curlLabel") }}
				</label>
				<input
					id="apiTokenCurl"
					:value="curlExample"
					type="text"
					class="form-control font-monospace"
					readonly
					@focus="selectInput"
				/>
				<CopyLink :text="curlExample" />
			</div>

			<div class="alert alert-warning mb-4">
				{{ $t("config.apitoken.warning") }}
			</div>

			<div class="d-flex justify-content-end">
				<button type="button" class="btn btn-primary" data-bs-dismiss="modal">
					{{ $t("config.apitoken.done") }}
				</button>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import PasswordInput from "../Auth/PasswordInput.vue";
import CopyLink from "../Helper/CopyLink.vue";
import api from "@/api";
import { docsPrefix } from "@/i18n";
import { getBaseUrl } from "@/utils/url";

export default defineComponent({
	name: "ApiTokenModal",
	components: { GenericModal, PasswordInput, CopyLink },
	data() {
		return {
			confirm: false,
			token: "",
			password: "",
			loading: false,
			error: "",
		};
	},
	computed: {
		docsUrl() {
			return `${docsPrefix()}/docs/integrations/rest-api`;
		},
		curlExample() {
			return `curl -H "Authorization: Bearer ${this.token}" ${getBaseUrl()}/api/config/devices/meter`;
		},
	},
	methods: {
		open() {
			this.resetState();
		},
		closed() {
			this.resetState();
		},
		resetState() {
			this.confirm = false;
			this.token = "";
			this.password = "";
			this.error = "";
			this.loading = false;
		},
		showConfirmStep() {
			this.confirm = true;
			this.error = "";
			this.$nextTick(() => {
				(this.$refs["passwordInput"] as any)?.$refs?.password?.focus();
			});
		},
		cancelConfirm() {
			this.confirm = false;
			this.password = "";
			this.error = "";
		},
		async generateToken() {
			if (!this.password) {
				return;
			}

			this.loading = true;
			this.error = "";

			try {
				const response = await api.post("/auth/token", {
					password: this.password,
				});
				this.token = response.data;
				this.password = "";
				this.confirm = false;
			} catch (error: any) {
				console.error(error);
				if (error.response?.status === 401 || error.response?.status === 400) {
					this.error = this.$t("loginModal.invalid");
				} else {
					this.error = error.response?.data || error.message;
				}
			} finally {
				this.loading = false;
			}
		},
		selectToken(event: Event) {
			const target = event.target as HTMLTextAreaElement;
			target.select();
		},
		selectInput(event: Event) {
			const target = event.target as HTMLInputElement;
			target.select();
		},
	},
});
</script>
