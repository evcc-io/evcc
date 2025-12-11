<template>
	<div>
		<FormRow
			id="messagingServiceTelegramToken"
			label="Token"
			:help="$t('config.messaging.telegram.token')"
		>
			<PropertyField
				id="messagingServiceTelegramToken"
				v-model="telegramOther.token"
				type="String"
				required
			/>
		</FormRow>
		<FormRow
			id="messagingServiceTelegramChats"
			label="Recipients"
			:help="$t('config.messaging.telegram.chats')"
		>
			<PropertyField
				id="messagingServiceTelegramChats"
				v-model="telegramOther.chats"
				property="chats"
				type="List"
				required
			/>
		</FormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceTelegram } from "@/types/evcc";
import FormRow from "../FormRow.vue";
import PropertyField from "../PropertyField.vue";
import type { PropType } from "vue";

export default {
	name: "TelegramService",
	components: { FormRow, PropertyField },
	props: {
		service: {
			type: Object as PropType<MessagingServiceTelegram>,
			required: true,
		},
	},
	computed: {
		telegramOther(): any {
			return this.service.other;
		},
	},
};
</script>
