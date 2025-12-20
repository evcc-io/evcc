<template>
	<div>
		<MessagingFormRow
			:serviceType="service.type"
			inputName="uri"
			example="gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1"
			:helpTranslationParameter="{
				url: '[containrrr.dev](https://containrrr.dev/shoutrrr/latest/services/overview/)',
			}"
		>
			<PropertyField
				id="messagingServiceShoutUri"
				v-model="serviceData.other.uri"
				type="String"
				required
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceShout } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import type { PropType } from "vue";
import MessagingFormRow from "./MessagingFormRow.vue";

export default {
	name: "ShoutService",
	components: { MessagingFormRow, PropertyField },
	props: {
		service: {
			type: Object as PropType<MessagingServiceShout>,
			required: true,
		},
	},
	data() {
		return {
			serviceData: this.service,
		};
	},
};
</script>
