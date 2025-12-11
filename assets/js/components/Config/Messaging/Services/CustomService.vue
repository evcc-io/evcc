<template>
	<div>
		<FormRow
			id="messagingServiceCustomEncoding"
			label="Encoding"
			:help="$t('config.messaging.custom.encoding')"
			optional
		>
			<PropertyField
				id="messagingServiceCustomEncoding"
				property="encoding"
				type="Choice"
				class="me-2 w-25"
				:choice="Object.values(MESSAGING_SERVICE_CUSTOM_ENCODING)"
				:model-value="service.other.encoding"
				@update:model-value="(e) => (service.other.encoding = e)"
			/>
		</FormRow>
		<YamlEntry v-model="service.other.send" type="messaging"></YamlEntry>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceCustom, MESSAGING_SERVICE_CUSTOM_ENCODING } from "@/types/evcc";
import type { PropType } from "vue";
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";
import YamlEntry from "../../DeviceModal/YamlEntry.vue";

export default {
	name: "CustomService",
	components: { FormRow, PropertyField, YamlEntry },
	props: {
		service: {
			type: Object as PropType<MessagingServiceCustom>,
			required: true,
		},
	},
	data() {
		return { MESSAGING_SERVICE_CUSTOM_ENCODING };
	},
};
</script>
