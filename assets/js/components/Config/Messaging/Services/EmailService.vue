<template>
	<MessagingFormRow
		v-for="field in fieldsWithValues"
		:key="field.name"
		:serviceType="serviceType"
		:inputName="field.name"
		:example="field.example"
	>
		<PropertyField
			:id="`messagingServiceEmail${field.name}`"
			:model-value="field.value"
			:type="field.type || 'String'"
			:masked="field.masked"
			:property="field.property"
			:rows="field.rows"
			required
			@update:model-value="$emit(`update:${field.name}`, $event)"
		/>
	</MessagingFormRow>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import type { PropType } from "vue";

const FIELDS = [
	{ name: "host", example: "smtp.example.com" },
	{ name: "port", example: "465" },
	{ name: "user", example: "john.doe" },
	{ name: "password", masked: true },
	{ name: "from", example: "john.doe@example.com" },
	{ name: "to", example: "recipient@example.com", type: "List", property: "topics", rows: 4 },
];

export default {
	name: "EmailService",
	components: { MessagingFormRow, PropertyField },
	props: {
		host: { type: String, required: true },
		port: { type: Number, required: true },
		user: { type: String, required: true },
		password: { type: String, required: true },
		from: { type: String, required: true },
		to: { type: Array as PropType<string[]>, required: true },
	},
	emits: FIELDS.map((f) => `update:${f.name}`),
	data() {
		return {
			serviceType: MESSAGING_SERVICE_TYPE.EMAIL,
		};
	},
	computed: {
		fieldsWithValues() {
			return FIELDS.map((field) => ({
				...field,
				value: (this as any)[field.name],
			}));
		},
	},
};
</script>
