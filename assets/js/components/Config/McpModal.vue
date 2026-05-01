<template>
	<GenericModal
		id="mcpModal"
		:title="`${$t('config.mcp.title')} 🧪`"
		config-modal-name="mcp"
		data-testid="mcp-modal"
	>
		<p>
			{{ $t("config.mcp.description") }}
		</p>
		<ErrorMessage :error="error" />
		<div class="form-check form-switch my-3">
			<input
				id="mcpEnabled"
				:checked="enabled"
				class="form-check-input"
				type="checkbox"
				role="switch"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="mcpEnabled">
					{{ $t("config.mcp.enable") }}
				</label>
			</div>
		</div>
		<p v-if="changed" class="text-muted small mt-2">
			{{ $t("config.mcp.restartHint") }}
		</p>
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
	name: "McpModal",
	components: { GenericModal, ErrorMessage },
	data() {
		return {
			error: null as string | null,
			changed: false,
		};
	},
	computed: {
		enabled(): boolean {
			return !!store.state?.mcp;
		},
	},
	methods: {
		async change(e: Event) {
			try {
				this.error = null;
				await api.post(`config/mcp/${(e.target as HTMLInputElement).checked}`);
				this.changed = true;
			} catch (err) {
				const e = err as AxiosError<{ error: string }>;
				this.error = e.response?.data?.error || e.message;
			}
		},
	},
});
</script>
