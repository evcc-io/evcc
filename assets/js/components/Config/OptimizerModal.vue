<template>
	<GenericModal
		id="optimizerModal"
		:title="$t('config.optimizer.title')"
		config-modal-name="optimizer"
		data-testid="optimizer-modal"
	>
		<p>
			{{ $t("config.optimizer.description") }}
			<a :href="docsLink" target="_blank">{{ $t("config.general.docsLink") }}</a>
		</p>
		<ErrorMessage :error="error" />
		<div class="form-check form-switch my-3">
			<input
				id="optimizerEnabled"
				:checked="enabled"
				class="form-check-input"
				type="checkbox"
				role="switch"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="optimizerEnabled">
					{{ $t("config.optimizer.enable") }}
				</label>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import ErrorMessage from "../Helper/ErrorMessage.vue";
import api from "@/api";
import store from "@/store";
import { docsPrefix } from "@/i18n";
import type { AxiosError } from "axios";

export default defineComponent({
	name: "OptimizerModal",
	components: { GenericModal, ErrorMessage },
	data() {
		return {
			error: null as string | null,
		};
	},
	computed: {
		enabled(): boolean {
			return !!store.state?.optimizer;
		},
		docsLink(): string {
			return `${docsPrefix()}/docs/features/optimizer`;
		},
	},
	methods: {
		async change(e: Event) {
			try {
				this.error = null;
				await api.post(`config/optimizer/${(e.target as HTMLInputElement).checked}`);
			} catch (err) {
				const e = err as AxiosError<{ error: string }>;
				this.error = e.response?.data?.error || e.message;
			}
		},
	},
});
</script>
