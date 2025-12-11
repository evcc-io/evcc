<template>
	<div>
		<FormRow
			id="messagingServiceNftyHost"
			label="Host"
			:help="$t('config.messaging.nfty.host')"
		>
			<PropertyField
				id="messagingServiceNftyHost"
				:model-value="decoded['host']"
				type="String"
				required
				@update:model-value="(e) => updateNfty('host', e)"
			/>
		</FormRow>
		<FormRow
			id="messagingServiceNftyTopics"
			label="Topics"
			:help="$t('config.messaging.nfty.topics')"
		>
			<PropertyField
				id="messagingServiceNftyTopics"
				:model-value="(decoded['topics'] ?? '').split(',')"
				property="topics"
				type="List"
				required
				@update:model-value="(e) => updateNfty('topics', e)"
			/>
		</FormRow>
		<FormRow
			id="messagingServiceNftyPriority"
			label="Priority"
			:help="$t('config.messaging.nfty.priority')"
			optional
		>
			<PropertyField
				id="messagingServiceNftyPriority"
				property="priority"
				type="Choice"
				class="me-2 w-25"
				:choice="Object.values(MESSAGING_SERVICE_NFTY_PRIORITY)"
				:model-value="nftyOther.priority"
				@update:model-value="(e) => (nftyOther.priority = e)"
			/>
		</FormRow>
		<FormRow
			id="messagingServiceNftyTags"
			label="Recipients"
			:help="$t('config.messaging.nfty.tags')"
			optional
		>
			<PropertyField
				id="messagingServiceNftyTags"
				:model-value="nftyOther.tags?.split(',')"
				property="tags"
				type="List"
				@update:model-value="(e: string[]) => (nftyOther.tags = e.join())"
			/>
		</FormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceNfty, MESSAGING_SERVICE_NFTY_PRIORITY } from "@/types/evcc";
import type { PropType } from "vue";
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";

export default {
	name: "NftyService",
	components: { FormRow, PropertyField },
	props: {
		service: {
			type: Object as PropType<MessagingServiceNfty>,
			required: true,
		},
	},
	data() {
		return { MESSAGING_SERVICE_NFTY_PRIORITY };
	},
	computed: {
		nftyOther(): any {
			return this.service.other;
		},
		decoded(): Record<string, string> {
			const nftyOther = this.service.other as any;
			let hostname = "";
			let pathname = "";

			try {
				const url = new URL(nftyOther.uri);
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
		updateNfty(p: string, v: string | string[]) {
			const nftyOther = this.service.other as any;
			const d = this.decoded;
			d[p] = Array.isArray(v) ? v.join(",") : v;
			nftyOther.uri = `https://${d["host"]}/${d["topics"]}`;
		},
	},
};
</script>
