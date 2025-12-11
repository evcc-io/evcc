<template>
	<div>
		<FormRow id="messagingServiceShoutUri" label="Uri" :help="$t('config.messaging.shout.uri')">
			<PropertyField
				id="messagingServiceShoutUri"
				v-model="service.other.uri"
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
};
</script>
