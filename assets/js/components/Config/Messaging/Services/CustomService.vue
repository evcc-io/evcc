<template>
	<div>
		<MessagingFormRow
			:serviceType="serviceType"
			inputName="encoding"
			optional
			:helpI18nParams="{
				send: '`${send}`',
				format: getEncodingFormat,
			}"
		>
			<PropertyField
				id="messagingServiceCustomEncoding"
				property="encoding"
				type="Choice"
				class="me-2 w-25"
				:choice="Object.values(MESSAGING_SERVICE_CUSTOM_ENCODING)"
				:model-value="encoding"
				@update:model-value="$emit('update:encoding', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="serviceType"
			inputName="send"
			:helpI18nParams="{
				url: '[docs.evcc.io](https://docs.evcc.io/en/docs/devices/plugins)',
			}"
		>
			<YamlEditorContainer :model-value="send" @update:model-value="updateSend" />
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_CUSTOM_ENCODING, MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import YamlEditorContainer from "../../YamlEditorContainer.vue";
import MessagingFormRow from "./MessagingFormRow.vue";

const DEAFULT_SEND_PLUGIN = `source: script
cmd: /usr/local/bin/evcc_message "{{.send}}"
`;

export default {
	name: "CustomService",
	components: { MessagingFormRow, PropertyField, YamlEditorContainer },
	props: {
		encoding: String as PropType<MESSAGING_SERVICE_CUSTOM_ENCODING>,
		send: String,
	},
	emits: ["update:encoding", "update:send"],
	data() {
		return {
			serviceType: MESSAGING_SERVICE_TYPE.CUSTOM,
			newContent: {},
			changed: false,
			MESSAGING_SERVICE_CUSTOM_ENCODING,
			DEAFULT_SEND_PLUGIN,
		};
	},
	computed: {
		getEncodingFormat() {
			switch (this.encoding) {
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
		if (!this.send) {
			this.updateSend(DEAFULT_SEND_PLUGIN);
		}
	},
	methods: {
		updateSend(v: string) {
			this.$emit("update:send", v);
		},
	},
};
</script>
