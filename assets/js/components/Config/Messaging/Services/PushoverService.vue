<template>
	<div>
		<MessagingFormRow
			:serviceType="service.type"
			inputName="app"
			example="azGDORePK8gMaC0QOYAMyEEuzJnyUi"
			:helpTranslationParameter="{
				url: '[pushover.net](https://pushover.net/apps/build)',
			}"
		>
			<PropertyField
				id="messagingServicePushoverApp"
				v-model="pushoverOther.app"
				type="String"
				required
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="service.type"
			inputName="recipients"
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
				rows
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="service.type" inputName="devices" example="droid2">
			<PropertyField
				id="messagingServicePushoverDevices"
				v-model="pushoverOther.devices"
				property="devices"
				type="List"
				required
				rows
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServicePushover } from "@/types/evcc";
import PropertyField from "../../PropertyField.vue";
import type { PropType } from "vue";
import MessagingFormRow from "./MessagingFormRow.vue";

export default {
	name: "PushoverService",
	components: { MessagingFormRow, PropertyField },
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
