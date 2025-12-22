<template>
	<div>
		<MessagingFormRow :serviceType="serviceType" inputName="host" example="ntfy.sh">
			<PropertyField
				id="messagingServiceNtfyHost"
				:model-value="host"
				type="String"
				required
				@update:model-value="$emit('update:host', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="serviceType" inputName="topics" example="evcc_alert">
			<PropertyField
				id="messagingServiceNtfyTopics"
				:model-value="topics"
				property="topics"
				type="List"
				required
				rows
				@update:model-value="$emit('update:topics', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="serviceType"
			inputName="authtoken"
			example="tk_7eevizlsiwf9yi4uxsrs83r4352o0"
			optional
		>
			<PropertyField
				id="messagingServiceNtfyAuthtoken"
				:model-value="authtoken"
				type="String"
				required
				@update:model-value="$emit('update:authtoken', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="serviceType" inputName="priority" optional>
			<PropertyField
				id="messagingServiceNtfyPriority"
				:model-value="priority"
				property="priority"
				type="Choice"
				class="me-2 w-25"
				:choice="Object.values(MESSAGING_SERVICE_NTFY_PRIORITY)"
				@update:model-value="$emit('update:priority', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="serviceType"
			inputName="tags"
			example="electric_plug,blue_car"
			optional
		>
			<PropertyField
				id="messagingServiceNtfyTags"
				:model-value="tags?.split(',')"
				property="tags"
				rows
				type="List"
				@update:model-value="$emit('update:tags', $event.join())"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_NTFY_PRIORITY, MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";

export default {
	name: "NtfyService",
	components: { MessagingFormRow, PropertyField },
	props: {
		host: {
			type: String,
			required: true,
		},
		topics: {
			type: Array as PropType<string[]>,
			required: true,
		},
		priority: String as PropType<MESSAGING_SERVICE_NTFY_PRIORITY>,
		tags: String,
		authtoken: String,
	},
	emits: ["update:host", "update:topics", "update:priority", "update:tags", "update:authtoken"],
	data() {
		return { serviceType: MESSAGING_SERVICE_TYPE.NTFY, MESSAGING_SERVICE_NTFY_PRIORITY };
	},
};
</script>
