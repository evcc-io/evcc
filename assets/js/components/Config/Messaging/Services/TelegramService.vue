<template>
	<div>
		<FormRow
			id="messagingServiceTelegramToken"
			:label="$t('config.messaging.service.telegram.token')"
			:help="$t('config.messaging.service.telegram.tokenHelp')"
			example="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
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
			:label="$t('config.messaging.service.telegram.chats')"
			:help="$t('config.messaging.service.telegram.chatsHelp')"
			example="-210987654"
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
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";
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
