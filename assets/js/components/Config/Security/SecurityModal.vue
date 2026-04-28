<template>
	<GenericModal
		id="securityModal"
		config-modal-name="security"
		:title="$t('config.security.title')"
		data-testid="security-modal"
		@open="onOpen"
	>
		<div v-if="authDisabled" class="alert alert-warning">
			{{ $t("config.security.authDisabledHint") }}
		</div>
		<p v-else>{{ $t("config.security.description") }}</p>

		<div class="mb-3">
			<h6>{{ $t("config.security.passwordTitle") }}</h6>
			<p>{{ $t("config.security.passwordDescription") }}</p>
			<button
				type="button"
				class="btn btn-outline-secondary"
				:disabled="authDisabled"
				@click="open('passwordupdate')"
			>
				{{ $t("config.security.updatePassword") }}
			</button>
		</div>

		<hr class="my-4" />

		<div class="mb-3">
			<h6>{{ $t("config.apiKey.title") }}</h6>
			<p>{{ $t("config.apiKey.entryDescription") }}</p>

			<template v-if="apiKeyConfigured">
				<input
					type="text"
					class="form-control font-monospace"
					:value="fakeKey"
					readonly
					aria-label="API Key"
				/>
				<div class="d-flex justify-content-end mt-2">
					<button
						type="button"
						class="btn btn-link btn-sm text-muted px-0"
						:disabled="authDisabled"
						@click="open('apikey')"
					>
						{{ $t("config.apiKey.regenerateLink") }}
					</button>
				</div>
			</template>

			<PlaceholderButton v-else :disabled="authDisabled" @click="open('apikey')">
				<shopicon-regular-plus class="me-1"></shopicon-regular-plus>
				<span>{{ $t("config.apiKey.generate") }}</span>
			</PlaceholderButton>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../../Helper/GenericModal.vue";
import PlaceholderButton from "../../Helper/PlaceholderButton.vue";
import { openModal } from "@/configModal";
import api from "@/api";
import "@h2d2/shopicons/es/regular/plus";

const FAKE_KEY = "evcc_" + "•".repeat(30);

export default defineComponent({
	name: "SecurityModal",
	components: { GenericModal, PlaceholderButton },
	props: {
		authDisabled: Boolean,
	},
	data() {
		return {
			apiKeyConfigured: false,
			fakeKey: FAKE_KEY,
		};
	},
	methods: {
		async onOpen() {
			try {
				const res = await api.get("auth/apikey");
				this.apiKeyConfigured = !!res.data?.configured;
			} catch {
				this.apiKeyConfigured = false;
			}
		},
		open(name: string) {
			if (this.authDisabled) return;
			openModal(name);
		},
	},
});
</script>
