<template>
	<div>
		<FormRow
			id="messagingServiceNtfyHost"
			:label="$t('config.messaging.service.ntfy.host')"
			:help="$t('config.messaging.service.ntfy.hostHelp')"
			example="ntfy.sh"
		>
			<PropertyField
				id="messagingServiceNtfyHost"
				:model-value="decoded['host']"
				type="String"
				required
				@update:model-value="(e) => updateNtfy('host', e)"
			/>
		</FormRow>
		<FormRow
			id="messagingServiceNtfyTopics"
			:label="$t('config.messaging.service.ntfy.topics')"
			:help="$t('config.messaging.service.ntfy.topicsHelp')"
			example="evcc_alert"
		>
			<PropertyField
				id="messagingServiceNtfyTopics"
				:model-value="(decoded['topics'] ?? '').split(',')"
				property="topics"
				type="List"
				required
				@update:model-value="(e) => updateNtfy('topics', e)"
			/>
		</FormRow>
		<FormRow
			id="messagingServiceNtfyPriority"
			:label="$t('config.messaging.service.ntfy.priority')"
			:help="
				$t('config.messaging.service.ntfy.priorityHelp', {
					url: '[docs.ntfy.sh](https://docs.ntfy.sh/publish/#message-priority)',
				})
			"
			optional
		>
			<PropertyField
				id="messagingServiceNtfyPriority"
				property="priority"
				type="Choice"
				class="me-2 w-25"
				:choice="Object.values(MESSAGING_SERVICE_NTFY_PRIORITY)"
				:model-value="ntfyOther.priority"
				@update:model-value="(e) => (ntfyOther.priority = e)"
			/>
		</FormRow>
		<FormRow
			id="messagingServiceNtfyTags"
			:label="$t('config.messaging.service.ntfy.tags')"
			:help="
				$t('config.messaging.service.ntfy.tagsHelp', {
					url: '[docs.ntfy.sh](https://docs.ntfy.sh/publish/#tags-emojis)',
				})
			"
			example="electric_plug,blue_car"
			optional
		>
			<PropertyField
				id="messagingServiceNtfyTags"
				:model-value="ntfyOther.tags?.split(',')"
				property="tags"
				type="List"
				@update:model-value="(e: string[]) => (ntfyOther.tags = e.join())"
			/>
		</FormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceNtfy, MESSAGING_SERVICE_NTFY_PRIORITY } from "@/types/evcc";
import type { PropType } from "vue";
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";

export default {
	name: "NtfyService",
	components: { FormRow, PropertyField },
	props: {
		service: {
			type: Object as PropType<MessagingServiceNtfy>,
			required: true,
		},
	},
	data() {
		return { MESSAGING_SERVICE_NTFY_PRIORITY };
	},
	computed: {
		ntfyOther(): any {
			return this.service.other;
		},
		decoded(): Record<string, string> {
			const ntfyOther = this.service.other as any;
			let hostname = "";
			let pathname = "";

			try {
				const url = new URL(ntfyOther.uri);
				hostname = url.hostname;
				pathname = url.pathname.replace("/", "");
			} catch (e) {
				console.warn(e);
			}

			return {
				host: hostname,
				topics: pathname,
			};
		},
	},
	methods: {
		updateNtfy(p: string, v: string | string[]) {
			const ntfyOther = this.service.other as any;
			const d = this.decoded;
			d[p] = Array.isArray(v) ? v.join(",") : v;
			ntfyOther.uri = `https://${d["host"]}/${d["topics"]}`;
		},
	},
};
</script>
