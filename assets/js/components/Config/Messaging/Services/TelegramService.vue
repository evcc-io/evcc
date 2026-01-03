<template>
	<div>
		<MessagingFormRow
			:serviceType="serviceType"
			inputName="token"
			example="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
		>
			<PropertyField
				id="messagingServiceTelegramToken"
				:model-value="token"
				type="String"
				required
				@update:model-value="$emit('update:token', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="serviceType" inputName="chats" example="-210987654">
			<PropertyField
				id="messagingServiceTelegramChats"
				:model-value="chats"
				property="chats"
				type="List"
				required
				rows
				@update:model-value="
					$emit(
						'update:chats',
						$event.map((v: string) => Number(v))
					)
				"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import type { PropType } from "vue";
import MessagingFormRow from "./MessagingFormRow.vue";

export default {
	name: "TelegramService",
	components: { MessagingFormRow, PropertyField },
	props: {
		token: {
			type: String,
			required: true,
		},
		chats: Array as PropType<number[]>,
	},
	emits: ["update:token", "update:chats"],
	data() {
		return { serviceType: MESSAGING_SERVICE_TYPE.TELEGRAM };
	},
};
</script>
