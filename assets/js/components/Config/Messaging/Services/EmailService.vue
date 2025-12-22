<template>
	<MessagingFormRow :serviceType="serviceType" inputName="host" example="emailserver.example.com">
		<PropertyField
			id="messagingServiceEmailHost"
			:model-value="host"
			type="String"
			required
			@update:model-value="$emit('update:host', $event)"
	/></MessagingFormRow>
	<MessagingFormRow :serviceType="serviceType" inputName="port" example="465">
		<PropertyField
			id="messagingServiceEmailPort"
			:model-value="port"
			type="String"
			required
			@update:model-value="$emit('update:port', $event)"
	/></MessagingFormRow>
	<MessagingFormRow :serviceType="serviceType" inputName="user" example="john.doe">
		<PropertyField
			id="messagingServiceEmailUser"
			:model-value="user"
			type="String"
			required
			@update:model-value="$emit('update:user', $event)"
	/></MessagingFormRow>
	<MessagingFormRow :serviceType="serviceType" inputName="password" example="secret123">
		<PropertyField
			id="messagingServiceEmailPassword"
			:model-value="password"
			type="String"
			required
			@update:model-value="$emit('update:password', $event)"
	/></MessagingFormRow>
	<MessagingFormRow
		:serviceType="serviceType"
		inputName="from"
		example="john.doe@mail.com"
		@update:model-value="$emit('update:from', $event)"
	>
		<PropertyField id="messagingServiceEmailFrom" :model-value="from" type="String" required
	/></MessagingFormRow>
	<MessagingFormRow
		:serviceType="serviceType"
		inputName="to"
		example="recipient@mail.com"
		@update:model-value="$emit('update:to', $event)"
	>
		<PropertyField id="messagingServiceEmailTo" :model-value="to" type="String" required
	/></MessagingFormRow>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import formatter from "@/mixins/formatter";

export default {
	name: "EmailService",
	components: { MessagingFormRow, PropertyField },
	mixins: [formatter],
	props: {
		host: {
			type: String,
			required: true,
		},
		port: {
			type: Number,
			required: true,
		},
		user: {
			type: String,
			required: true,
		},
		password: {
			type: String,
			required: true,
		},
		from: {
			type: String,
			required: true,
		},
		to: {
			type: String,
			required: true,
		},
	},
	emits: [
		"update:host",
		"update:port",
		"update:user",
		"update:password",
		"update:from",
		"update:to",
	],
	data() {
		return {
			serviceType: MESSAGING_SERVICE_TYPE.EMAIL,
		};
	},
};
</script>
