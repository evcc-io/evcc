<template>
	<div>
		<MessagingFormRow :serviceType="service.type" inputName="host" example="ntfy.sh">
			<PropertyField
				id="messagingServiceNtfyHost"
				v-model="serviceData.other.host"
				type="String"
				required
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="service.type" inputName="topics" example="evcc_alert">
			<PropertyField
				id="messagingServiceNtfyTopics"
				v-model="serviceData.other.topics"
				property="topics"
				type="List"
				required
				rows
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
				v-model="serviceData.other.authtoken"
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
				v-model="serviceData.other.priority"
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
				:model-value="serviceData.other.tags?.split(',')"
				property="tags"
				rows
				type="List"
				@update:model-value="(e: string[]) => (serviceData.other.tags = e.join())"
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
		return { serviceData: this.service, MESSAGING_SERVICE_NTFY_PRIORITY };
	},
};
</script>
