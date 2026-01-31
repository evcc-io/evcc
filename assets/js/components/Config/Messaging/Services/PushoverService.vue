<template>
	<div>
		<MessagingFormRow
			v-for="field in fieldsWithValues"
			:id="field.id"
			:key="field.name"
			:serviceType="serviceType"
			:inputName="field.name"
			:example="field.example"
			:optional="!field.required"
			:helpI18nParams="field.helpI18nParams"
		>
			<PropertyField
				:id="field.id"
				:model-value="field.value"
				:type="field.type || 'String'"
				:masked="field.masked"
				:rows="field.rows"
				:size="field.size"
				:class="field.class"
				:required="field.required"
				@update:model-value="$emit(`update:${field.name}`, $event)"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import type { PropType } from "vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import { formId } from "../utils";

const FIELDS = [
	{
		name: "app",
		example: "azGDORePK8gMaC0QOYAMyEEuzJnyUi",
		masked: true,
		required: true,
		helpI18nParams: { url: "[pushover.net](https://pushover.net/apps/build)" },
	},
	{
		name: "recipients",
		example: "uQiRzpo4DXghDmr9QzzfQu27cmVRsG",
		type: "List",
		rows: 4,
		size: "w-100",
		class: "me-2",
		required: true,
		helpI18nParams: { url: "[pushover.net](https://pushover.net/groups/build)" },
	},
	{ name: "devices", example: "droid2", type: "List", rows: 4 },
];

export default {
	name: "PushoverService",
	components: { MessagingFormRow, PropertyField },
	props: {
		app: { type: String, required: true },
		recipients: { type: Array as PropType<string[]>, required: true },
		devices: { type: Array as PropType<string[]>, required: true },
	},
	emits: FIELDS.map((f) => `update:${f.name}`),
	computed: {
		serviceType() {
			return MESSAGING_SERVICE_TYPE.PUSHOVER;
		},
		fieldsWithValues() {
			return FIELDS.map((field) => ({
				...field,
				id: formId(this.serviceType, field.name),
				value: (this as any)[field.name],
			}));
		},
	},
	methods: {
		formId(name: string) {
			return formId(this.serviceType, name);
		},
	},
};
</script>
