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
				:model-value="serviceData.other.encoding"
				@update:model-value="(e) => (serviceData.other.encoding = e)"
			/>
		</FormRow>
		<FormRow
			id="messagingServiceCustomSend"
			:label="$t('config.messaging.service.custom.send')"
			:help="
				$t('config.messaging.service.custom.sendHelp', {
					url: '[docs.evcc.io](https://docs.evcc.io/en/docs/devices/plugins#plugins-1)',
				})
			"
		>
			<YamlEditorContainer v-model="serviceData.other.send" />
		</FormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceCustom, MESSAGING_SERVICE_CUSTOM_ENCODING } from "@/types/evcc";
import type { PropType } from "vue";
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";
import YamlEditorContainer from "../../YamlEditorContainer.vue";

const DEAFULT_SEND_PLUGIN = 'source: script\ncmd: /usr/local/bin/evcc_message "{{.send}}"';

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
		return {
			serviceData: this.service,
			MESSAGING_SERVICE_CUSTOM_ENCODING,
			DEAFULT_SEND_PLUGIN,
		};
	},
	computed: {
		getEncodingFormat() {
			switch (this.serviceData.other.encoding) {
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
		if (!this.serviceData.other.send) {
			this.serviceData.other.send = DEAFULT_SEND_PLUGIN;
		}
	},
};
</script>
