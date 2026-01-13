<template>
	<div>
		<MessagingFormRow
			:id="formId('token')"
			:serviceType="serviceType"
			inputName="token"
			example="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
		>
			<PropertyField
				:id="formId('token')"
				masked
				:model-value="token"
				type="String"
				required
				@update:model-value="$emit('update:token', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:id="formId('chats')"
			:serviceType="serviceType"
			inputName="chats"
			example="-210987654"
		>
			<PropertyField
				:id="formId('chats')"
				:model-value="chats"
				type="List"
				required
				:rows="4"
				@update:model-value="$emit('update:chats', $event)"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import type { PropType } from "vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import { formId } from "../utils";

export default {
	name: "TelegramService",
	components: { MessagingFormRow, PropertyField },
	props: {
		token: { type: String, required: true },
		chats: Array as PropType<number[]>,
	},
	emits: ["update:token", "update:chats"],
	computed: {
		serviceType() {
			return MESSAGING_SERVICE_TYPE.TELEGRAM;
		},
	},
	methods: {
		formId(name: string) {
			return formId(this.serviceType, name);
		},
	},
};
</script>
