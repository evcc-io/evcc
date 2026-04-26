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

		<div class="mb-3">
			<h6>{{ $t("config.apiKey.title") }}</h6>
			<p>{{ $t("config.apiKey.entryDescription") }}</p>
			<button
				type="button"
				class="btn btn-outline-secondary"
				:disabled="authDisabled"
				@click="open('apikey')"
			>
				{{
					apiKeyConfigured ? $t("config.apiKey.regenerate") : $t("config.apiKey.generate")
				}}
			</button>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../../Helper/GenericModal.vue";
import { replaceModal } from "@/configModal";
import api from "@/api";

export default defineComponent({
	name: "SecurityModal",
	components: { GenericModal },
	props: {
		authDisabled: Boolean,
	},
	data() {
		return {
			apiKeyConfigured: false,
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
			replaceModal(name);
		},
	},
});
</script>
