<template>
	<GenericModal
		id="mcpModal"
		:title="`${$t('config.mcp.title')} 🧪`"
		config-modal-name="mcp"
		data-testid="mcp-modal"
	>
		<div v-if="!mcpActive" class="alert alert-warning mb-4" role="alert">
			{{ $t("config.mcp.restartHint") }}
		</div>
		<p>
			{{ $t("config.mcp.description") }}
			<a :href="docsLink" target="_blank">{{ $t("config.general.docsLink") }}</a>
		</p>
		<FormRow id="mcpModalServerUrl" :label="$t('config.mcp.url')">
			<input
				id="mcpModalServerUrl"
				type="text"
				class="form-control border"
				:value="mcpUrl"
				readonly
			/>
			<CopyLink :text="mcpUrl" />
		</FormRow>
		<FormRow id="mcpModalExample" :label="$t('config.mcp.exampleLabel')">
			<pre
				id="mcpModalExample"
				class="form-control border font-monospace small mb-2 mcp-example"
				>{{ claudeExample }}</pre
			>
			<CopyLink :text="claudeExample" />
		</FormRow>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import CopyLink from "../Helper/CopyLink.vue";
import FormRow from "./FormRow.vue";
import store from "@/store";
import { docsPrefix } from "@/i18n";

export default defineComponent({
	name: "McpModal",
	components: { GenericModal, CopyLink, FormRow },
	computed: {
		mcpActive(): boolean {
			return !!store.state?.mcp;
		},
		mcpUrl(): string {
			return `${window.location.origin}/mcp`;
		},
		claudeExample(): string {
			return `claude mcp add --transport http evcc ${this.mcpUrl}`;
		},
		docsLink(): string {
			return `${docsPrefix()}/docs/integrations/mcp`;
		},
	},
});
</script>

<style scoped>
.mcp-example {
	white-space: pre;
	overflow-x: scroll;
	width: 100%;
	box-sizing: border-box;
}
</style>
