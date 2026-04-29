<template>
	<GenericModal
		id="optimizerModal"
		:title="`${$t('config.optimizer.title')} 🧪`"
		config-modal-name="optimizer"
		data-testid="optimizer-modal"
	>
		<p>
			{{ $t("config.optimizer.description") }}
			<a :href="docsLink" target="_blank">{{ $t("config.general.docsLink") }}</a>
		</p>
		<SponsorTokenRequired v-if="!isSponsor" feature class="mt-0" />
		<ErrorMessage :error="error" />
		<div class="form-check form-switch my-3">
			<input
				id="optimizerEnabled"
				:checked="enabled"
				class="form-check-input"
				type="checkbox"
				role="switch"
				:disabled="!isSponsor"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="optimizerEnabled">
					{{ $t("config.optimizer.enable") }}
				</label>
			</div>
		</div>
		<p v-if="enabled && !hasEvopt" class="text-muted small mt-2">
			{{ $t("config.optimizer.info") }}
		</p>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import ErrorMessage from "../Helper/ErrorMessage.vue";
import SponsorTokenRequired from "./DeviceModal/SponsorTokenRequired.vue";
import api from "@/api";
import store from "@/store";
import { docsPrefix } from "@/i18n";
import type { AxiosError } from "axios";

export default defineComponent({
	name: "OptimizerModal",
	components: { GenericModal, ErrorMessage, SponsorTokenRequired },
	props: {
		isSponsor: Boolean,
	},
	data() {
		return {
			error: null as string | null,
		};
	},
	computed: {
		enabled(): boolean {
			return !!store.state?.optimizer;
		},
		hasEvopt(): boolean {
			return !!store.state?.evopt;
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
