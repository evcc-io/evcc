<template>
	<div>
		<MessagingFormRow :serviceType="service.type" inputName="host" example="ntfy.sh">
			<PropertyField
				id="messagingServiceNtfyHost"
				:model-value="decoded['host']"
				type="String"
				required
				@update:model-value="(e) => updateNtfy('host', e)"
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="service.type" inputName="topics" example="evcc_alert">
			<PropertyField
				id="messagingServiceNtfyTopics"
				:model-value="(decoded['topics'] ?? '').split(',')"
				property="topics"
				type="List"
				required
				rows
				@update:model-value="(e) => updateNtfy('topics', e)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="service.type"
			inputName="authtoken"
			example="tk_7eevizlsiwf9yi4uxsrs83r4352o0"
			optional
		>
			<PropertyField
				id="messagingServiceNtfyAuthtoken"
				v-model="ntfyOther.authtoken"
				type="String"
				required
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="service.type" inputName="priority" optional>
			<PropertyField
				id="messagingServiceNtfyPriority"
				property="priority"
				type="Choice"
				class="me-2 w-25"
				:choice="Object.values(MESSAGING_SERVICE_NTFY_PRIORITY)"
				:model-value="ntfyOther.priority"
				@update:model-value="(e) => (ntfyOther.priority = e)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="service.type"
			inputName="tags"
			example="electric_plug,blue_car"
			optional
		>
			<PropertyField
				id="messagingServiceNtfyTags"
				:model-value="ntfyOther.tags?.split(',')"
				property="tags"
				rows
				type="List"
				@update:model-value="(e: string[]) => (ntfyOther.tags = e.join())"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceNtfy, MESSAGING_SERVICE_NTFY_PRIORITY } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";

export default {
	name: "NtfyService",
	components: { MessagingFormRow, PropertyField },
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
		ntfyOther() {
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
