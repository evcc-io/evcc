<template>
	<div>
		<FormRow
			id="messagingServiceCustomEncoding"
			:label="$t('config.messaging.service.custom.encoding')"
			:help="
				$t('config.messaging.service.custom.encodingHelp', {
					send: '`${send}`',
					format: getEncodingFormat,
				})
			"
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
		<FormRow
			id="messagingServiceCustomSend"
			:label="$t('config.messaging.service.custom.send')"
			:help="$t('config.messaging.service.custom.sendHelp')"
		>
			<YamlEditorContainer v-model="service.other.send" />
		</FormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceCustom, MESSAGING_SERVICE_CUSTOM_ENCODING } from "@/types/evcc";
import type { PropType } from "vue";
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";
import YamlEditorContainer from "../../YamlEditorContainer.vue";

const DEAFULT_SEND_PLUGIN = `send:
    source: script
    cmd: /usr/local/bin/evcc_message "{{.send}}"
`;

export default {
	name: "CustomService",
	components: { FormRow, PropertyField, YamlEditorContainer },
	props: {
		service: {
			type: Object as PropType<MessagingServiceCustom>,
			required: true,
		},
	},
	data() {
		return { MESSAGING_SERVICE_CUSTOM_ENCODING, DEAFULT_SEND_PLUGIN };
	},
	computed: {
		getEncodingFormat() {
			switch (this.service.other.encoding) {
				case MESSAGING_SERVICE_CUSTOM_ENCODING.JSON:
					return '`{ "msg": <MSG>, "title": <TITLE> }`';
				case MESSAGING_SERVICE_CUSTOM_ENCODING.CSV:
					return "`<TITLE>,<MSG>`";
				case MESSAGING_SERVICE_CUSTOM_ENCODING.TITLE:
					return "`<TITLE>`";
				case MESSAGING_SERVICE_CUSTOM_ENCODING.TSV:
					return "`<TITLE><TAB><MSG>`";
				default:
					return "`<MSG>`";
			}
		},
	},
	mounted() {
		if (!this.service.other.send) {
			this.service.other.send = DEAFULT_SEND_PLUGIN;
		}
	},
};
</script>
