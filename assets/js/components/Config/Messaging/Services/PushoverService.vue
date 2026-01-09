<template>
	<div>
		<MessagingFormRow
			:serviceType="serviceType"
			inputName="app"
			example="azGDORePK8gMaC0QOYAMyEEuzJnyUi"
			:helpI18nParams="{
				url: '[pushover.net](https://pushover.net/apps/build)',
			}"
		>
			<PropertyField
				id="messagingServicePushoverApp"
				:model-value="app"
				type="String"
				required
				@update:model-value="$emit('update:app', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="serviceType"
			inputName="recipients"
			example="uQiRzpo4DXghDmr9QzzfQu27cmVRsG"
			:helpI18nParams="{
				url: '[pushover.net](https://pushover.net/groups/build)',
			}"
		>
			<PropertyField
				id="messagingServicePushoverRecipients"
				:model-value="recipients"
				property="recipients"
				type="List"
				size="w-100"
				class="me-2"
				:rows="4"
				@update:model-value="$emit('update:recipients', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow :serviceType="serviceType" inputName="devices" example="droid2" optional>
			<PropertyField
				id="messagingServicePushoverDevices"
				:model-value="devices"
				property="devices"
				type="List"
				:rows="4"
				@update:model-value="$emit('update:devices', $event)"
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
	name: "PushoverService",
	components: { MessagingFormRow, PropertyField },
	props: {
		app: {
			type: String,
			required: true,
		},
		recipients: {
			type: Array as PropType<string[]>,
			required: true,
		},
		devices: {
			type: Array as PropType<string[]>,
			required: true,
		},
	},
	emits: ["update:app", "update:recipients", "update:devices"],
	data() {
		return { serviceType: MESSAGING_SERVICE_TYPE.PUSHOVER };
	},
};
</script>
