<template>
	<div>
		<FormRow
			id="messagingServiceShoutUri"
			:label="$t('config.messaging.service.shout.uri')"
			:help="
				$t('config.messaging.service.shout.uriHelp', {
					url: '[containrrr.dev](https://containrrr.dev/shoutrrr/latest/services/overview/)',
				})
			"
			example="gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1"
		>
			<PropertyField
				id="messagingServiceShoutUri"
				v-model="serviceData.other.uri"
				type="String"
				required
			/>
		</FormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceShout } from "@/types/evcc";
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";
import type { PropType } from "vue";

export default {
	name: "ShoutService",
	components: { FormRow, PropertyField },
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
