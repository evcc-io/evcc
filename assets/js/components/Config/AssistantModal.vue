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
					@change="loadModels(values)"
				/>
			</FormRow>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import api from "@/api";
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
		return {
			providers: Object.keys(PROVIDERS) as AssistantProvider[],
			// models the configured endpoint reports, empty until it answers
			offered: [] as string[],
			// endpoint the offered models belong to, so a change refetches
			offeredFor: "",
		};
	},
	methods: {
		withDefaults(values?: AssistantConfig): AssistantConfig {
			const res: AssistantConfig = values?.provider
				? values
				: { ...values, provider: "openai", model: PROVIDERS.openai.models[0] };

			// the field offers what arrived by the time it is focused, a datalist
			// does not open once options are added to it
			this.loadModels(res);

			return res;
		},
		needsToken(values: AssistantConfig): boolean {
			return values.provider !== "ollama";
		},
		models(values: AssistantConfig): string[] {
			const suggested = PROVIDERS[values.provider]?.models || [];
			// the curated ones lead, an endpoint also lists image and embedding models
			return [...suggested, ...this.offered.filter((m) => !suggested.includes(m))];
		},
		// loadModels asks the endpoint what it offers, silently keeping the
		// suggestions when it cannot be reached or needs a token first
		async loadModels(values: AssistantConfig) {
			const key = `${values.provider}|${values.baseUrl || ""}`;
			if (key === this.offeredFor) return;
			this.offeredFor = key;
			this.offered = [];

			try {
				const res = await api.post("/assistant/models", values);
				if (Array.isArray(res.data)) this.offered = res.data;
			} catch {
				// no suggestions from the endpoint, the curated list remains
			}
		},
		baseUrlExample(values: AssistantConfig): string {
			return PROVIDERS[values.provider]?.baseUrl || "";
		},
		applyDefaults(values: AssistantConfig) {
			this.offered = [];
			this.offeredFor = "";

			const provider = PROVIDERS[values.provider];
			if (!provider) return;
			// keep a hand-typed model, replace a suggestion that belongs to another provider
			if (!values.model || SUGGESTED.includes(values.model)) {
				values.model = provider.models[0] || "";
			}
			if (!values.baseUrl) values.baseUrl = provider.baseUrl;

			this.loadModels(values);
		},
	},
});
</script>
