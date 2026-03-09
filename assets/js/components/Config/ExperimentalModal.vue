<template>
	<GenericModal
		id="experimentalModal"
		:title="`${$t('config.experimental.title')} 🧪`"
		config-modal-name="experimental"
		data-testid="experimental-modal"
	>
		<p>{{ $t("config.experimental.description") }}</p>
		<ErrorMessage :error="error" />
		<div class="form-check form-switch my-3">
			<input
				id="experimentalEnabled"
				:checked="experimental"
				class="form-check-input"
				type="checkbox"
				role="switch"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="experimentalEnabled">
					{{ $t("settings.hiddenFeatures.value") }}
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
import type { AxiosError } from "axios";

export default defineComponent({
	name: "ExperimentalModal",
	components: { GenericModal, ErrorMessage },
	props: {
		experimental: Boolean,
	},
	data() {
		return {
			error: null as string | null,
		};
	},
	methods: {
		async change(e: Event) {
			try {
				this.error = null;
				await api.post(`config/experimental/${(e.target as HTMLInputElement).checked}`);
			} catch (err) {
				const e = err as AxiosError<{ error: string }>;
				this.error = e.response?.data?.error || e.message;
			}
		},
	},
});
</script>
