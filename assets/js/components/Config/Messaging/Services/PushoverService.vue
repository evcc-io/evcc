<template>
	<div>
		<FormRow
			id="messagingServicePushoverApp"
			:label="$t('config.messaging.service.pushover.app')"
			:help="
				$t('config.messaging.service.pushover.appHelp', {
					url: '[pushover.net](https://pushover.net/apps/build)',
				})
			"
			example="azGDORePK8gMaC0QOYAMyEEuzJnyUi"
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
			:label="$t('config.messaging.service.pushover.recipients')"
			:help="$t('config.messaging.service.pushover.recipientsHelp')"
			example="uQiRzpo4DXghDmr9QzzfQu27cmVRsG"
		>
			<PropertyField
				id="messagingServicePushoverRecipients"
				v-model="pushoverOther.recipients"
				property="recipients"
				type="List"
				size="w-100"
				class="me-2"
				required
			/>
		</FormRow>
		<FormRow
			id="messagingServicePushoverDevices"
			:label="$t('config.messaging.service.pushover.devices')"
			:help="$t('config.messaging.service.pushover.devicesHelp')"
			example="droid2"
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
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";
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
		pushoverOther() {
			return this.service.other;
		},
	},
};
</script>
