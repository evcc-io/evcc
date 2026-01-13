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
				:class="field.class"
				:choice="field.choice"
				:required="field.required"
				@update:model-value="$emit(`update:${field.name}`, field.updateTransform($event))"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_NTFY_PRIORITY, MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import { formId } from "../utils";

const FIELDS = [
	{ name: "host", example: "ntfy.sh", required: true },
	{ name: "topics", example: "evcc_alert", type: "List", rows: 4, required: true },
	{ name: "authtoken", example: "tk_7eevizlsiwf9yi4uxsrs83r4352o0", masked: true },
	{
		name: "priority",
		type: "Choice",
		class: "me-2 w-25",
		choice: Object.values(MESSAGING_SERVICE_NTFY_PRIORITY),
		helpI18nParams: { url: "[docs.ntfy.sh](https://docs.ntfy.sh/publish#message-priority)" },
	},
	{
		name: "tags",
		example: "electric_plug,blue_car",
		type: "List",
		rows: 4,
		helpI18nParams: { url: "[docs.ntfy.sh](https://docs.ntfy.sh/publish#tags-emojis)" },
		valueTransform: (value: string) => value?.split(","),
		updateTransform: (event: string[]) => event.join(),
	},
];

export default {
	name: "NtfyService",
	components: { MessagingFormRow, PropertyField },
	props: {
		host: { type: String, required: true },
		topics: { type: Array as PropType<string[]>, required: true },
		priority: String as PropType<MESSAGING_SERVICE_NTFY_PRIORITY>,
		tags: String,
		authtoken: String,
	},
	emits: FIELDS.map((f) => `update:${f.name}`),
	computed: {
		serviceType() {
			return MESSAGING_SERVICE_TYPE.NTFY;
		},
		fieldsWithValues() {
			return FIELDS.map((field: any) => {
				const rawValue = (this as any)[field.name];
				const value = field.valueTransform ? field.valueTransform(rawValue) : rawValue;
				const updateTransform = field.updateTransform || ((v: any) => v);

				return {
					...field,
					id: formId(this.serviceType, field.name),
					value,
					updateTransform,
				};
			});
		},
	},
	methods: {
		formId(name: string) {
			return formId(this.serviceType, name);
		},
	},
};
</script>
