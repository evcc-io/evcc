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
				:help="$t('config.assistant.helpModel')"
			>
				<input
					id="assistantModel"
					v-model="values.model"
					class="form-control"
					list="assistantModels"
					required
				/>
				<datalist id="assistantModels">
					<option v-for="model in models(values)" :key="model" :value="model" />
				</datalist>
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

// suggested models per provider, first entry is the default. free text, any model may be entered
const PROVIDERS: Record<AssistantProvider, { models: string[]; baseUrl: string }> = {
	openai: { models: ["gpt-5", "gpt-5-mini", "gpt-4.1", "gpt-4o-mini"], baseUrl: "" },
	anthropic: {
		models: ["claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"],
		baseUrl: "",
	},
	ollama: {
		models: [
			"qwen3",
			"llama3.1",
			"llama3.2",
			"mistral-nemo",
			"gpt-oss",
			"command-r7b",
			"granite3.3",
			"firefunction-v2",
		],
		baseUrl: "http://localhost:11434",
	},
	custom: { models: [], baseUrl: "https://generativelanguage.googleapis.com/v1beta/openai" },
};

const SUGGESTED = Object.values(PROVIDERS).flatMap(({ models }) => models);

export default defineComponent({
	name: "AssistantModal",
	components: { FormRow, JsonModal },
	emits: ["changed"],
	data() {
		return { providers: Object.keys(PROVIDERS) as AssistantProvider[] };
	},
	methods: {
		withDefaults(values?: AssistantConfig): AssistantConfig {
			if (values?.provider) return values;
			return { ...values, provider: "openai", model: PROVIDERS.openai.models[0] };
		},
		needsToken(values: AssistantConfig): boolean {
			return values.provider !== "ollama";
		},
		models(values: AssistantConfig): string[] {
			return PROVIDERS[values.provider]?.models || [];
		},
		baseUrlExample(values: AssistantConfig): string {
			return PROVIDERS[values.provider]?.baseUrl || "";
		},
		applyDefaults(values: AssistantConfig) {
			const provider = PROVIDERS[values.provider];
			if (!provider) return;
			// keep a hand-typed model, replace a suggestion that belongs to another provider
			if (!values.model || SUGGESTED.includes(values.model)) {
				values.model = provider.models[0] || "";
			}
			if (!values.baseUrl) values.baseUrl = provider.baseUrl;
		},
	},
});
</script>
