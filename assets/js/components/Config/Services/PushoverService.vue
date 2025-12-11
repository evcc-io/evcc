<template>
	<div>
		<FormRow
			id="messagingServicePushoverApp"
			label="App"
			:help="$t('config.messaging.pushover.app')"
		>
			<PropertyField
				id="messagingServicePushoverApp"
				v-model="pushoverOther.app"
				type="String"
				required
			/>
		</FormRow>
		<FormRow
			id="messagingServicePushoverRecipients"
			label="Recipients"
			:help="$t('config.messaging.pushover.recipients')"
		>
			<PropertyField
				id="messagingServicePushoverRecipients"
				v-model="pushoverOther.devices"
				property="recipients"
				type="List"
				size="w-100"
				class="me-2"
				required
			/>
		</FormRow>
		<FormRow
			id="messagingServicePushoverDevices"
			label="Devices"
			:help="$t('config.messaging.pushover.devices')"
		>
			<PropertyField
				id="messagingServicePushoverDevices"
				v-model="pushoverOther.devices"
				property="devices"
				type="List"
				required
			/>
		</FormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServicePushover } from "@/types/evcc";
import FormRow from "../FormRow.vue";
import PropertyField from "../PropertyField.vue";
import type { PropType } from "vue";

export default {
	name: "PushoverService",
	components: { FormRow, PropertyField },
	props: {
		service: {
			type: Object as PropType<MessagingServicePushover>,
			required: true,
		},
	},
	computed: {
		pushoverOther(): any {
			return this.service.other;
		},
	},
};
</script>
