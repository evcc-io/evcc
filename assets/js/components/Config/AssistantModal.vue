<template>
	<JsonModal
		name="assistant"
		:title="`${$t('config.assistant.title')} 🧪`"
		:description="$t('config.assistant.description')"
		endpoint="/config/assistant"
		state-key="assistant"
		:transform-read-values="withDefaults"
		@changed="$emit('changed')"
	>
		<template #default="{ values }">
			<FormRow id="assistantProvider" :label="$t('config.assistant.labelProvider')">
				<select
					id="assistantProvider"
					v-model="values.provider"
					class="form-select"
					required
					@change="applyDefaults(values)"
				>
					<option v-for="p in providers" :key="p" :value="p">
						{{ $t(`config.assistant.provider.${p}`) }}
					</option>
				</select>
			</FormRow>
			<FormRow
				id="assistantModel"
				:label="$t('config.assistant.labelModel')"
				:example="modelExample(values)"
			>
				<input id="assistantModel" v-model="values.model" class="form-control" required />
			</FormRow>
			<FormRow
				v-if="needsToken(values)"
				id="assistantToken"
				:label="$t('config.assistant.labelToken')"
			>
				<input
					id="assistantToken"
					v-model="values.token"
					class="form-control"
					type="password"
					autocomplete="off"
					required
				/>
			</FormRow>
			<FormRow
				id="assistantBaseUrl"
				:label="$t('config.assistant.labelBaseUrl')"
				:help="$t('config.assistant.helpBaseUrl')"
				:example="baseUrlExample(values)"
				:optional="values.provider !== 'custom'"
			>
				<input
					id="assistantBaseUrl"
					v-model="values.baseUrl"
					type="url"
					class="form-control"
					:required="values.provider === 'custom'"
				/>
			</FormRow>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import type { AssistantConfig, AssistantProvider } from "@/types/evcc";

const DEFAULTS: Record<AssistantProvider, { model: string; baseUrl: string }> = {
	openai: { model: "gpt-4o-mini", baseUrl: "" },
	anthropic: { model: "claude-sonnet-4-5", baseUrl: "" },
	ollama: { model: "qwen3", baseUrl: "http://localhost:11434" },
	custom: { model: "", baseUrl: "https://generativelanguage.googleapis.com/v1beta/openai" },
};

export default defineComponent({
	name: "AssistantModal",
	components: { FormRow, JsonModal },
	emits: ["changed"],
	data() {
		return { providers: Object.keys(DEFAULTS) as AssistantProvider[] };
	},
	methods: {
		withDefaults(values?: AssistantConfig): AssistantConfig {
			if (values?.provider) return values;
			return { ...values, provider: "openai", model: DEFAULTS.openai.model };
		},
		needsToken(values: AssistantConfig): boolean {
			return values.provider !== "ollama";
		},
		modelExample(values: AssistantConfig): string {
			return DEFAULTS[values.provider]?.model || "";
		},
		baseUrlExample(values: AssistantConfig): string {
			return DEFAULTS[values.provider]?.baseUrl || "";
		},
		applyDefaults(values: AssistantConfig) {
			const defaults = DEFAULTS[values.provider];
			if (!defaults) return;
			if (!values.model) values.model = defaults.model;
			if (!values.baseUrl) values.baseUrl = defaults.baseUrl;
		},
	},
});
</script>
