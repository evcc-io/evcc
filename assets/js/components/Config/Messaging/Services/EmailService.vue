<template>
	<MessagingFormRow
		:serviceType="service.type"
		inputName="host"
		example="emailserver.example.com"
	>
		<PropertyField
			id="messagingServiceEmailHost"
			v-model="serviceData.other.host"
			type="String"
			required
	/></MessagingFormRow>
	<MessagingFormRow :serviceType="service.type" inputName="port" example="587">
		<PropertyField
			id="messagingServiceEmailPort"
			v-model="serviceData.other.port"
			type="String"
			required
	/></MessagingFormRow>
	<MessagingFormRow :serviceType="service.type" inputName="user" example="john.doe">
		<PropertyField
			id="messagingServiceEmailUser"
			v-model="serviceData.other.user"
			type="String"
			required
	/></MessagingFormRow>
	<MessagingFormRow :serviceType="service.type" inputName="password" example="secret123">
		<PropertyField
			id="messagingServiceEmailPassword"
			v-model="serviceData.other.password"
			type="String"
			required
	/></MessagingFormRow>
	<MessagingFormRow :serviceType="service.type" inputName="from" example="john.doe@mail.com">
		<PropertyField
			id="messagingServiceEmailFrom"
			v-model="serviceData.other.from"
			type="String"
			required
	/></MessagingFormRow>
	<MessagingFormRow :serviceType="service.type" inputName="to" example="recipient@mail.com">
		<PropertyField
			id="messagingServiceEmailTo"
			v-model="serviceData.other.to"
			type="String"
			required
	/></MessagingFormRow>
</template>

<script lang="ts">
import { type MessagingServiceEmail } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import formatter from "@/mixins/formatter";

export default {
	name: "EmailService",
	components: { MessagingFormRow, PropertyField },
	mixins: [formatter],
	props: {
		service: {
			type: Object as PropType<MessagingServiceEmail>,
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
