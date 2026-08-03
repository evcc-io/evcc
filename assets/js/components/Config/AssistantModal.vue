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
					<option v-for="p in providers" :key="p.provider" :value="p.provider">
						{{ $t(`config.assistant.provider.${p.provider}`) }}
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
					class="form-control"
					list="assistantModels"
					required
					:value="values.model"
					@input="values.model = ($event.target as HTMLInputElement).value"
					@focus="clearWhilePicking"
					@blur="restoreModel(values, $event)"
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
				:optional="!needsBaseUrl(values)"
			>
				<input
					id="assistantBaseUrl"
					v-model="values.baseUrl"
					type="url"
					class="form-control"
					:required="needsBaseUrl(values)"
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
import type { AssistantConfig, AssistantProviderInfo } from "@/types/evcc";

export default defineComponent({
	name: "AssistantModal",
	components: { FormRow, JsonModal },
	emits: ["changed"],
	data() {
		return {
			// the backend owns the provider list, its defaults and its mandatory fields
			providers: [] as AssistantProviderInfo[],
			// values being edited, so defaults can be applied once the list arrived
			editing: null as AssistantConfig | null,
			// models the configured endpoint reports, empty until it answers
			offered: [] as string[],
			// endpoint the offered models belong to, so a change refetches
			offeredFor: "",
		};
	},
	created() {
		this.loadProviders();
	},
	methods: {
		async loadProviders() {
			const res = await api.get("/assistant/providers");
			if (!Array.isArray(res.data)) return;
			this.providers = res.data;
			// the config may have been read before the list arrived
			if (this.editing && !this.editing.model) this.applyDefaults(this.editing);
		},
		info(values: AssistantConfig): AssistantProviderInfo | undefined {
			return this.providers.find((p) => p.provider === values.provider);
		},
		withDefaults(values?: AssistantConfig): AssistantConfig {
			const res: AssistantConfig = values?.provider
				? values
				: { ...values, provider: "openai", model: "" };
			this.editing = res;

			if (!res.model) this.applyDefaults(res);

			// the field offers what arrived by the time it is focused, a datalist
			// does not open once options are added to it
			this.loadModels(res);

			return res;
		},
		needsToken(values: AssistantConfig): boolean {
			return this.info(values)?.needsToken ?? true;
		},
		needsBaseUrl(values: AssistantConfig): boolean {
			return this.info(values)?.needsBaseUrl ?? false;
		},
		models(values: AssistantConfig): string[] {
			const suggested = this.info(values)?.models || [];
			// the curated ones lead, an endpoint also lists image and embedding models
			return [...suggested, ...this.offered.filter((m) => !suggested.includes(m))];
		},
		// a datalist only offers the options matching what is in the field, so the
		// value steps aside while picking and comes back when nothing was chosen
		clearWhilePicking(event: FocusEvent) {
			(event.target as HTMLInputElement).value = "";
		},
		restoreModel(values: AssistantConfig, event: FocusEvent) {
			const input = event.target as HTMLInputElement;
			if (!input.value) input.value = values.model || "";
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
			return this.info(values)?.baseUrl || "";
		},
		applyDefaults(values: AssistantConfig) {
			this.offered = [];
			this.offeredFor = "";

			const info = this.info(values);
			if (!info) return;
			// keep a hand-typed model, replace a suggestion that belongs to another provider
			const suggested = this.providers.flatMap((p) => p.models || []);
			if (!values.model || suggested.includes(values.model)) {
				values.model = info.models?.[0] || "";
			}
			if (!values.baseUrl) values.baseUrl = info.baseUrl || "";

			this.loadModels(values);
		},
	},
});
</script>
