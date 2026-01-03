<template>
	<div>
		<MessagingFormRow
			:serviceType="MESSAGING_SERVICE_TYPE.SHOUT"
			inputName="uri"
			example="gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1"
			:helpTranslationParameter="{
				url: '[containrrr.dev](https://containrrr.dev/shoutrrr/latest/services/overview/)',
			}"
		>
			<PropertyField
				id="messagingServiceShoutUri"
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

export default {
	name: "ShoutService",
	components: { MessagingFormRow, PropertyField },
	props: {
		uri: {
			type: String,
			required: true,
		},
	},
	emits: ["update:uri"],
	data() {
		return {
			MESSAGING_SERVICE_TYPE,
		};
	},
};
</script>
