<template>
	<div>
		<MessagingFormRow
			:serviceType="service.type"
			inputName="token"
			example="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
		>
			<PropertyField
				id="messagingServiceTelegramToken"
				v-model="telegramOther.token"
				type="String"
				required
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="service.type" inputName="chats" example="-210987654">
			<PropertyField
				id="messagingServiceTelegramChats"
				v-model="telegramOther.chats"
				property="chats"
				type="List"
				required
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceTelegram } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import type { PropType } from "vue";
import MessagingFormRow from "./MessagingFormRow.vue";

export default {
	name: "TelegramService",
	components: { MessagingFormRow, PropertyField },
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
