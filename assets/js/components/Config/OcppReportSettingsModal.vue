<template>
	<GenericModal
		id="ocppReportSettingsModal"
		:title="`${$t('config.ocppreportsettings.title')} 🧪`"
		config-modal-name="ocppreportsettings"
		data-testid="ocppreportsettings-modal"
	>
		<p>{{ $t("config.ocppreportsettings.description") }}</p>
		<ErrorMessage :error="error" />
		<div class="form-check form-switch my-3">
			<input
				id="ocppReportEnabled"
				:checked="enabled"
				class="form-check-input"
				type="checkbox"
				role="switch"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="ocppReportEnabled">
					{{ $t("config.ocppreportsettings.enableLabel") }}
				</label>
			</div>
		</div>
		<p class="text-muted small mb-0">{{ ruleCountLabel }}</p>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import ErrorMessage from "../Helper/ErrorMessage.vue";
import api from "@/api";
import store from "@/store";
import type { AxiosError } from "axios";

export default defineComponent({
	name: "OcppReportSettingsModal",
	components: { GenericModal, ErrorMessage },
	props: {
		enabled: Boolean,
	},
	data() {
		return {
			error: null as string | null,
		};
	},
	computed: {
		ruleCount(): number {
			return store.state?.ocppreport?.config?.length || 0;
		},
		ruleCountLabel(): string {
			// @ts-expect-error plural
			return this.$t("config.ocppreportsettings.ruleCount", this.ruleCount, {
				count: this.ruleCount,
			});
		},
	},
	methods: {
		async change(e: Event) {
			try {
				this.error = null;
				const value = (e.target as HTMLInputElement).checked;
				await api.post(`config/ocppreportenabled/${value}`);
			} catch (err) {
				const e = err as AxiosError<{ error: string }>;
				this.error = e.response?.data?.error || e.message;
			}
		},
	},
});
</script>
