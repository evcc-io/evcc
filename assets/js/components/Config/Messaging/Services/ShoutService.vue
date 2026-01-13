<template>
	<div>
		<MessagingFormRow
			:id="formId('uri')"
			:serviceType="serviceType"
			inputName="uri"
			example="gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1"
			:helpI18nParams="{
				url: '[containrrr.dev](https://containrrr.dev/shoutrrr/latest/services/overview/)',
			}"
		>
			<PropertyField
				:id="formId('uri')"
				:model-value="uri"
				type="String"
				required
				@update:model-value="$emit('update:uri', $event)"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import { formId } from "../utils";

export default {
	name: "ShoutService",
	components: { MessagingFormRow, PropertyField },
	props: {
		uri: { type: String, required: true },
	},
	emits: ["update:uri"],
	computed: {
		serviceType() {
			return MESSAGING_SERVICE_TYPE.SHOUT;
		},
	},
	methods: {
		formId(name: string) {
			return formId(this.serviceType, name);
		},
	},
};
</script>
